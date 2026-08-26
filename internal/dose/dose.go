package dose

import (
	"fmt"
	"sync"
	"time"

	"waterplant/internal/pump"
)

type Command struct {
	ID     string
	Source string
	Amount float64
	At     time.Time
}

type Doser struct {
	mu          sync.Mutex
	pump        *pump.Pump
	flow        FlowWaiter
	plan        *Plan
	flowFn      func() float64
	flowTimeout time.Duration
	state       State
	executed    map[string]bool
	count       int
	last        Command
	fault       string
}

func New(pump *pump.Pump, flow FlowWaiter, plan *Plan, flowFn func() float64, flowTimeout time.Duration) *Doser {
	return &Doser{
		pump:        pump,
		flow:        flow,
		plan:        plan,
		flowFn:      flowFn,
		flowTimeout: flowTimeout,
		state:       StateIdle,
		executed:    make(map[string]bool),
	}
}

func (d *Doser) Start(cmd Command) error {
	d.mu.Lock()
	if d.executed[cmd.ID] {
		d.mu.Unlock()
		return nil
	}
	d.mu.Unlock()
	if err := d.flow.Wait(d.flowFn, d.flowTimeout); err != nil {
		return fmt.Errorf("flow wait: %w", err)
	}
	// 启泵失败必须如实反映:不要吞错后还把状态置为 dosing,
	// 否则中控会显示投加中,而现场泵并未转,余氯持续走低。
	if err := d.pump.Start(); err != nil {
		d.mu.Lock()
		d.state = StateFault
		d.fault = "pump start: " + err.Error()
		d.mu.Unlock()
		return fmt.Errorf("pump start: %w", err)
	}
	d.mu.Lock()
	d.executed[cmd.ID] = true
	d.count++
	d.last = cmd
	d.state = StateDosing
	d.fault = ""
	d.mu.Unlock()
	return nil
}

func (d *Doser) Stop(reason string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	// 停泵是显式操作:投加中要停,故障态(可能停泵指令失败、泵仍在转)也要强制再下一次停令,
	// 避免中控显示已停而现场泵还在转。空闲态无可停,直接返回。
	if d.state == StateIdle {
		return nil
	}
	if err := d.pump.Stop(); err != nil {
		d.state = StateFault
		d.fault = reason + ": " + err.Error()
		return err
	}
	d.state = StateIdle
	d.fault = ""
	return nil
}

func (d *Doser) Snapshot() Snapshot {
	d.mu.Lock()
	defer d.mu.Unlock()
	return Snapshot{
		State:    d.state,
		LastID:   d.last.ID,
		Executed: d.count,
		Amount:   d.last.Amount,
		Fault:    d.fault,
		At:       d.last.At,
	}
}

// Restart 供值班员在启动失败(fault)后手动重新下发启动。
// 它先尝试把泵从故障态复位(Stop),再重新 Start;与自动 EvaluateDosing 不同,
// 它不走冷却限流——现场余氯走低时必须给值班员一个能立即重试的入口。
func (d *Doser) Restart(cmd Command) error {
	d.mu.Lock()
	if d.state == StateDosing {
		d.mu.Unlock()
		return nil
	}
	d.state = StateIdle
	d.fault = ""
	d.mu.Unlock()
	// 故障态下泵需要先复位再启动;Stop 在 stopped/fault 态会复位 driver,
	// 失败也不阻塞——真正的判定在随后的 Start 中以运行反馈为准。
	_ = d.pump.Stop()
	return d.Start(cmd)
}
