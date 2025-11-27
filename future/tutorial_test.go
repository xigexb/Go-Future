package future_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/xigexb/go-future/future"
)

// 对应教程第 1 节：快速开始
func TestGuide_QuickStart(t *testing.T) {
	fmt.Println("=== Test 1: Quick Start ===")

	f := future.SupplyAsync(func() int {
		time.Sleep(100 * time.Millisecond)
		return 100
	})

	result, err := f.Join()

	if err != nil {
		t.Fatalf("任务出错: %v", err)
	}
	if result != 100 {
		t.Errorf("预期 100, 实际 %d", result)
	}
	fmt.Printf("计算结果: %d\n", result)
}

// 对应教程第 2 节：链式转换
func TestGuide_Chaining(t *testing.T) {
	fmt.Println("\n=== Test 2: Chaining ===")

	f1 := future.SupplyAsync(func() int {
		return 10
	})

	f2 := future.ThenApply(f1, func(v int) string {
		return fmt.Sprintf("订单号: %d", v)
	})

	f3 := future.ThenApply(f2, func(s string) string {
		return s + " 已发货"
	})

	f3.ThenAccept(func(s string) {
		fmt.Println("最终处理结果:", s)
	})

	res, err := f3.Join()
	if err != nil {
		t.Fatal(err)
	}
	expected := "订单号: 10 已发货"
	if res != expected {
		t.Errorf("预期 '%s', 实际 '%s'", expected, res)
	}
}

// 对应教程第 3 节：AllOf / AnyOf
func TestGuide_Combination(t *testing.T) {
	fmt.Println("\n=== Test 3: Combination (AllOf/AnyOf) ===")

	f1 := future.SupplyAsync(func() string { return "InfoA" })
	f2 := future.SupplyAsync(func() string { return "InfoB" })

	all := future.AllOf(f1, f2)
	_, err := all.Join()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("AllOf 完成")

	fSlow := future.SupplyAsync(func() string {
		time.Sleep(100 * time.Millisecond)
		return "Slow"
	})
	fFast := future.SupplyAsync(func() string {
		time.Sleep(10 * time.Millisecond)
		return "Fast"
	})

	anyF := future.AnyOf(fSlow, fFast)
	res, _ := anyF.Join()
	fmt.Println("最快的源是:", res)

	if res != "Fast" {
		t.Errorf("AnyOf 应该返回 Fast, 实际返回 %s", res)
	}
}

// 对应教程第 4 节：异常处理
func TestGuide_Exception(t *testing.T) {
	fmt.Println("\n=== Test 4: Exception Handling ===")

	f := future.SupplyAsync(func() int {
		panic("糟糕，数据库挂了！") // 模拟崩溃
		return 0
	})

	// 自动捕获并恢复
	// 修改点：返回 (T, error)
	safeFuture := f.Exceptionally(func(err error) (int, error) {
		fmt.Printf("捕获到错误: %v\n", err)
		return -1, nil // nil 表示成功恢复
	})

	res, _ := safeFuture.Join()
	if res != -1 {
		t.Errorf("Exception recovery failed, got %d", res)
	}
	fmt.Println("恢复后的结果:", res)
}

// 对应教程第 5 节：超时控制
func TestGuide_Timeout(t *testing.T) {
	fmt.Println("\n=== Test 5: Timeout Control ===")

	f := future.SupplyAsync(func() string {
		time.Sleep(200 * time.Millisecond) // 模拟慢任务
		return "ok"
	})

	f.OrTimeout(50 * time.Millisecond)

	_, err := f.Join()
	if err == future.ErrTimeout {
		fmt.Println("成功捕获任务超时！")
	} else {
		t.Errorf("预期超时错误, 实际得到: %v", err)
	}
}
