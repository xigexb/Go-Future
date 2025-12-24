package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/xigexb/go-future/pool"

	"github.com/xigexb/go-future/future"
)

type User struct {
	ID   int
	Name string
	VIP  bool
}

type Order struct {
	OrderID string
	Amount  float64
}

func main() {
	// 1. 初始化自定义线程池（可选，展示 SetGlobalExecutor 或传入 Executor）
	// 这里我们创建一个用于处理耗时IO的专用池
	ioExecutor := pool.NewBlockingExecutor(10)
	fmt.Println("=== 订单处理系统启动 ===")
	start := time.Now()

	// ==========================================
	// 场景一：并行获取基础数据 (SupplyAsync)
	// ==========================================

	// 任务A: 获取用户信息 (模拟 500ms)
	futureUser := future.SupplyAsync(func() User {
		fmt.Println("[任务A] 正在获取用户信息...")
		time.Sleep(500 * time.Millisecond)
		return User{ID: 101, Name: "张三", VIP: true}
	})

	// 任务B: 获取购物车总价 (模拟 300ms)
	futureAmount := future.SupplyAsync(func() float64 {
		fmt.Println("[任务B] 正在计算购物车金额...")
		time.Sleep(300 * time.Millisecond)
		return 1000.00
	})

	// ==========================================
	// 场景二：组合两个独立任务的结果 (ThenCombine)
	// 注意：你的库中 ThenCombine 是函数形式
	// ==========================================

	futureOrder := future.ThenCombine(futureUser, futureAmount, func(u User, amount float64) Order {
		fmt.Printf("[任务C] 合并数据: 用户[%s] 原始金额[%.2f]\n", u.Name, amount)
		finalAmount := amount
		if u.VIP {
			finalAmount = amount * 0.8 // VIP 打8折
			fmt.Println("[任务C] 应用VIP折扣")
		}
		return Order{OrderID: "ORD-20250101", Amount: finalAmount}
	})

	// ==========================================
	// 场景三：异步编排与依赖 (ThenCompose)
	// 获取订单后，发起支付（支付本身也是个异步过程）
	// ==========================================

	futurePayment := future.ThenCompose(futureOrder, func(o Order) *future.CompletableFuture[string] {
		// 这里返回一个新的 Future
		return future.SupplyAsyncWithExecutor(ioExecutor, func() string {
			fmt.Printf("[任务D] 开始支付处理: 单号 %s, 金额 %.2f\n", o.OrderID, o.Amount)

			// 模拟支付可能失败
			if rand.Intn(10) > 8 {
				panic("第三方支付网关无响应")
			}
			time.Sleep(400 * time.Millisecond)
			return "PAY-SUCCESS-" + o.OrderID
		})
	})

	// ==========================================
	// 场景四：异常处理与兜底 (Exceptionally / Handle)
	// ==========================================

	futureSafePayment := futurePayment.Handle(func(payId string, err error) string {
		if err != nil {
			fmt.Printf("[异常处理] 支付环节出错: %v，降级为'待人工审核'\n", err)
			return "MANUAL-CHECK-REQ" // 降级处理
		}
		return payId
	})

	// ==========================================
	// 场景五：超时控制 (OrTimeout / CompleteOnTimeout)
	// 如果支付环节整体超过 2秒（模拟），则强制超时
	// ==========================================

	// 为了演示超时，我们可以给它加上一个极短的超时时间，或者保留足够时间
	// 这里设置 2秒，正常应该能跑完
	futureWithTimeout := futureSafePayment.OrTimeout(2 * time.Second)

	// ==========================================
	// 场景六：并行竞争 (ApplyToEither)
	// 支付成功后，同时询问两家物流公司，谁回得快用谁
	// ==========================================

	futureLogistics := future.ThenCompose(futureWithTimeout, func(payResult string) *future.CompletableFuture[string] {
		if payResult == "MANUAL-CHECK-REQ" {
			return future.CompletedFuture("无需物流(审核中)")
		}

		// 物流A
		f1 := future.SupplyAsync(func() string {
			time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
			return "顺丰速运"
		})
		// 物流B
		f2 := future.SupplyAsync(func() string {
			time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
			return "京东物流"
		})

		// 谁先回来用谁，并拼接结果
		return future.ApplyToEither(f1, f2, func(provider string) string {
			fmt.Printf("[任务E] 选定物流商: %s\n", provider)
			return fmt.Sprintf("订单[%s] 已发货 via %s", payResult, provider)
		})
	})

	// ==========================================
	// 场景七：全任务聚合 (AllOf)
	// 比如：发送邮件通知 和 更新库存 必须都完成
	// ==========================================

	finalTask := future.ThenCompose(futureLogistics, func(finalStatus string) *future.CompletableFuture[struct{}] {
		fmt.Println("[最终阶段] 收到状态:", finalStatus)

		// 任务F: 发邮件 (无返回值)
		fEmail := future.RunAsync(func() {
			time.Sleep(100 * time.Millisecond)
			fmt.Println("[任务F] 邮件通知已发送")
		})

		// 任务G: 扣减库存
		fStock := future.RunAsync(func() {
			time.Sleep(150 * time.Millisecond)
			fmt.Println("[任务G] 库存已扣减")
		})

		// 等待两者都完成
		return future.AllOf(fEmail, fStock)
	})

	// ==========================================
	// 场景八：上下文控制 (Context)
	// 演示由外部 Context 取消的情况
	// ==========================================
	ctx, cancel := context.WithCancel(context.Background())
	fCancelDemo := future.SupplyAsyncCtx(ctx, func() int {
		time.Sleep(1 * time.Second)
		return 100
	})
	// 模拟中途取消
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// ==========================================
	// 最终执行与结果获取 (Join)
	// ==========================================

	// 等待主流程完成
	_, err := finalTask.Join()
	if err != nil {
		fmt.Printf("主流程异常结束: %v\n", err)
	} else {
		fmt.Println(">>> 主流程全部成功完成 <<<")
	}

	fmt.Printf("总耗时: %v\n", time.Since(start))

	// 检查那个被 Cancel 的任务
	_, err = fCancelDemo.Join()
	fmt.Printf("演示Cancel的任务状态: %v\n", err) // 应该输出 context canceled
}
