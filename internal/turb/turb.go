package turb

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"waterplant/internal/alarm"
	"waterplant/internal/dose"
	"waterplant/internal/event"
	"waterplant/internal/filter"
	"waterplant/internal/record"
)

var (
	ErrBadPayload           = errors.New("event payload type mismatch")
	ErrUnknownQualitySource = errors.New("unknown quality source")
)

type Controller struct {
	store           *Store
	assessor        *Assessor
	pool            *filter.Pool
	doser           *dose.Doser
	alarms          *alarm.Manager
	bus             *event.Bus
	recorder        *record.Recorder
	journal         *event.Journal
	plan            *dose.Plan
	limits          *dose.Limits
	cooldown        *dose.Cooldown
	snapshotPath    string
	backwashTimeout time.Duration
	lastSeq         map[string]uint64
	staleEvents     uint64
	seqMu           sync.Mutex
	stop            chan struct{}
}

func NewController(
	store *Store,
	assessor *Assessor,
	pool *filter.Pool,
	doser *dose.Doser,
	alarms *alarm.Manager,
	bus *event.Bus,
	recorder *record.Recorder,
	journal *event.Journal,
	plan *dose.Plan,
	limits *dose.Limits,
	cooldown *dose.Cooldown,
	snapshotPath string,
	backwashTimeout time.Duration,
) *Controller {
	return &Controller{
		store:           store,
		assessor:        assessor,
		pool:            pool,
		doser:           doser,
		alarms:          alarms,
		bus:             bus,
		recorder:        recorder,
		journal:         journal,
		plan:            plan,
		limits:          limits,
		cooldown:        cooldown,
		snapshotPath:    snapshotPath,
		backwashTimeout: backwashTimeout,
		lastSeq:         make(map[string]uint64),
		stop:            make(chan struct{}),
	}
}

func (c *Controller) Start() {
	ch, _ := c.bus.Subscribe(event.KindAlarm, 64)
	go func() {
		for {
			select {
			case ev := <-ch:
				_ = c.HandleEvent(ev)
			case <-c.stop:
				return
			}
		}
	}()
}

func (c *Controller) Stop() {
	close(c.stop)
}

func (c *Controller) ApplyQuality(q Quality) error {
	c.store.Put(q)
	if _, err := c.alarms.Evaluate(q.ID, "turbidity", q.Turbidity); err != nil {
		return err
	}
	if _, err := c.alarms.Evaluate(q.ID, "chlorine", q.Chlorine); err != nil {
		return err
	}
	_, err := c.bus.Publish(event.KindQuality, q.ID, q)
	return err
}

func (c *Controller) EvaluateDosing(source string) error {
	q, ok := c.store.Latest(source)
	if !ok {
		return ErrUnknownQualitySource
	}
	if !c.cooldown.Allow(source, time.Now()) {
		return nil
	}
	decision := c.assessor.Evaluate(q)
	if !decision.Dosing {
		return nil
	}
	c.cooldown.Mark(source, time.Now())
	cmd := dose.Command{
		ID:     fmt.Sprintf("auto-%d", time.Now().UnixNano()),
		Source: source,
		Amount: c.limits.Clamp(c.plan.AmountFor(q.Chlorine)),
		At:     time.Now(),
	}
	if err := c.StartDosing(cmd); err != nil {
		// 启动失败:保持冷却,避免自动回路在故障态下反复打火;
		// 值班员可通过 dose_restart 入口手动重新下发启动(绕过冷却)。
		return err
	}
	return nil
}

func (c *Controller) StartDosing(cmd dose.Command) error {
	if err := c.doser.Start(cmd); err != nil {
		// 启泵失败如实反映:记录失败、上报故障事件,不要发"投加成功"误导值班员。
		_ = c.recordFailure(cmd, err)
		_, _ = c.bus.Publish(event.KindDose, cmd.Source, map[string]any{
			"command": cmd,
			"ok":       false,
			"error":    err.Error(),
		})
		return fmt.Errorf("start dosing: %w", err)
	}
	_, _ = c.bus.Publish(event.KindDose, cmd.Source, cmd)
	return c.recordDose(cmd)
}

// RestartDosing 供值班员在投加泵启动失败后从中控手动重新下发启动。
// 它绕过自动冷却限流,确保余氯走低的紧急情况下能立即重试。
func (c *Controller) RestartDosing(source string) error {
	q, ok := c.store.Latest(source)
	if !ok {
		return ErrUnknownQualitySource
	}
	// 手动重新下发:清除该来源冷却,允许立即再次评估补投。
	c.cooldown.Reset(source)
	cmd := dose.Command{
		ID:     fmt.Sprintf("manual-%d", time.Now().UnixNano()),
		Source: source,
		Amount: c.limits.Clamp(c.plan.AmountFor(q.Chlorine)),
		At:     time.Now(),
	}
	if err := c.doser.Restart(cmd); err != nil {
		_ = c.recordFailure(cmd, err)
		_, _ = c.bus.Publish(event.KindDose, cmd.Source, map[string]any{
			"command": cmd,
			"ok":       false,
			"error":    err.Error(),
		})
		return fmt.Errorf("restart dosing: %w", err)
	}
	// 重新启动成功后补记冷却,避免与自动回路叠加重复投加。
	c.cooldown.Mark(source, time.Now())
	_, _ = c.bus.Publish(event.KindDose, cmd.Source, cmd)
	return c.recordDose(cmd)
}

func (c *Controller) Backwash(id string, confirm <-chan bool) error {
	if err := c.pool.BeginBackwash(id); err != nil {
		return err
	}
	select {
	case ok := <-confirm:
		if !ok {
			return c.pool.CancelBackwash(id)
		}
		if err := c.pool.ConfirmBackwash(id); err != nil {
			return err
		}
		if rec, found := c.pool.RecordFor(id); found {
			_, _ = c.bus.Publish(event.KindBackwash, id, rec)
		}
		return nil
	case <-time.After(c.backwashTimeout):
		return c.abortBackwash(id, filter.ErrPressureTimeout)
	}
}

func (c *Controller) abortBackwash(id string, cause error) error {
	if err := c.pool.CancelBackwash(id); err != nil {
		return err
	}
	_, _ = c.bus.Publish(event.KindBackwash, id, cause.Error())
	entry := record.Entry{
		At:     time.Now(),
		Source: id,
		Alarm:  "backwash-abort:" + cause.Error(),
	}
	_ = c.recorder.Append(entry)
	return cause
}

func (c *Controller) HandleEvent(ev event.Event) error {
	c.seqMu.Lock()
	last := c.lastSeq[ev.Subject]
	if ev.Seq <= last {
		c.staleEvents++
		c.seqMu.Unlock()
		return nil
	}
	c.lastSeq[ev.Subject] = ev.Seq
	c.seqMu.Unlock()
	switch ev.Kind {
	case event.KindAlarm:
		var st alarm.State
		switch payload := ev.Payload.(type) {
		case alarm.State:
			st = payload
		case json.RawMessage:
			if err := json.Unmarshal(payload, &st); err != nil {
				return ErrBadPayload
			}
		default:
			return ErrBadPayload
		}
		return c.alarms.ApplyEvent(ev.Subject, st)
	case event.KindQuality:
		var q Quality
		switch payload := ev.Payload.(type) {
		case Quality:
			q = payload
		case json.RawMessage:
			if err := json.Unmarshal(payload, &q); err != nil {
				return ErrBadPayload
			}
		default:
			return ErrBadPayload
		}
		c.store.Put(q)
		return nil
	default:
		return nil
	}
}

func (c *Controller) StaleEvents() uint64 {
	c.seqMu.Lock()
	defer c.seqMu.Unlock()
	return c.staleEvents
}

func (c *Controller) recordDose(cmd dose.Command) error {
	entry := record.Entry{
		At:     time.Now(),
		Source: cmd.Source,
		Alarm:  "dose",
	}
	if q, ok := c.store.Latest(cmd.Source); ok {
		entry.Turbidity = q.Turbidity
		entry.Chlorine = q.Chlorine
	}
	return c.recorder.Append(entry)
}

func (c *Controller) recordFailure(cmd dose.Command, cause error) error {
	entry := record.Entry{
		At:     time.Now(),
		Source: cmd.Source,
		Alarm:  "dose-failed:" + cause.Error(),
	}
	if q, ok := c.store.Latest(cmd.Source); ok {
		entry.Turbidity = q.Turbidity
		entry.Chlorine = q.Chlorine
	}
	return c.recorder.Append(entry)
}
