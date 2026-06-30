package workflow

import (
	"context"
	"os"
	"testing"
	"time"

	"encore.app/bill/domain"
	"github.com/oklog/ulid/v2"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const taskQueue = "test_task_queue"

var (
	testClient client.Client
	testWorker worker.Worker
)

func TestMain(m *testing.M) {
	var err error
	testClient, err = client.Dial(client.Options{
		HostPort:  "localhost:7233",
		Namespace: "default",
	})
	if err != nil {
		panic("failed to connect to Temporal: " + err.Error())
	}

	testWorker = worker.New(testClient, taskQueue, worker.Options{})
	testWorker.RegisterWorkflow(BillWorkflow)

	if err := testWorker.Start(); err != nil {
		panic(err)
	}

	code := m.Run()

	testWorker.Stop()
	testClient.Close()
	os.Exit(code)
}

func startTestBill(t *testing.T, ctx context.Context, currency domain.Currency) client.WorkflowRun {
	t.Helper()

	bwp := &BillWorkflowParams{
		Currency: currency,
		BillID:   ulid.Make().String(),
	}

	run, err := testClient.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		TaskQueue: taskQueue,
	}, BillWorkflow, bwp)
	if err != nil {
		t.Fatalf("failed to start workflow: %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	return run
}

func sendAddLineItem(ctx context.Context, run client.WorkflowRun, li domain.LineItem) (domain.LineItem, error) {
	handle, err := testClient.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   run.GetID(),
		RunID:        run.GetRunID(),
		UpdateName:   AddLineItem,
		Args:         []interface{}{li},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		return domain.LineItem{}, err
	}

	var result domain.LineItem
	if err := handle.Get(ctx, &result); err != nil {
		return domain.LineItem{}, err
	}
	return result, nil
}

func sendCloseBill(ctx context.Context, run client.WorkflowRun) (BillInvoice, error) {
	handle, err := testClient.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   run.GetID(),
		RunID:        run.GetRunID(),
		UpdateName:   CloseBill,
		Args:         []interface{}{},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		return BillInvoice{}, err
	}

	var result BillInvoice
	if err := handle.Get(ctx, &result); err != nil {
		return BillInvoice{}, err
	}
	return result, nil
}

func TestBillWorkflow_AddLineItem(t *testing.T) {
	tests := []struct {
		name         string
		itemCurrency domain.Currency
		wantErr      bool
	}{
		{
			name:         "accepts line item with matching currency",
			itemCurrency: domain.USD,
			wantErr:      false,
		},
		{
			name:         "rejects line item with mismatched currency",
			itemCurrency: domain.GEL,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			run := startTestBill(t, ctx, domain.USD)

			li, err := domain.NewLineItem("cheeseburger", 500, tt.itemCurrency)
			if err != nil {
				t.Fatalf("failed to construct line item: %v", err)
			}

			_, err = sendAddLineItem(ctx, run, *li)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if _, err := sendCloseBill(ctx, run); err != nil {
				t.Logf("cleanup close failed (non-fatal): %v", err)
			}
		})
	}
}

func TestBillWorkflow_AddLineItem_RejectedWhenClosed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	run := startTestBill(t, ctx, domain.USD)

	if _, err := sendCloseBill(ctx, run); err != nil {
		t.Fatalf("failed to close bill: %v", err)
	}

	li, err := domain.NewLineItem("late item", 500, domain.USD)
	if err != nil {
		t.Fatalf("failed to construct line item: %v", err)
	}

	_, err = sendAddLineItem(ctx, run, *li)
	if err == nil {
		t.Fatal("expected error when adding line item to closed bill, got nil")
	}
}

func TestBillWorkflow_MultipleLineItems_SumCorrectly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	run := startTestBill(t, ctx, domain.USD)

	amounts := []int64{500, 1200, 300}
	var wantTotal int64
	for _, amt := range amounts {
		wantTotal += amt
		li, err := domain.NewLineItem("item", amt, domain.USD)
		if err != nil {
			t.Fatalf("failed to construct line item: %v", err)
		}
		if _, err := sendAddLineItem(ctx, run, *li); err != nil {
			t.Fatalf("failed to add line item: %v", err)
		}
	}

	invoice, err := sendCloseBill(ctx, run)
	if err != nil {
		t.Fatalf("failed to close bill: %v", err)
	}

	if invoice.TotalInvoice != wantTotal {
		t.Errorf("TotalInvoice = %d, want %d", invoice.TotalInvoice, wantTotal)
	}
	if len(invoice.LineItems) != len(amounts) {
		t.Errorf("LineItems count = %d, want %d", len(invoice.LineItems), len(amounts))
	}
}

func TestBillWorkflow_AddLineItem_IdempotentOnDuplicateID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	run := startTestBill(t, ctx, domain.USD)

	li, err := domain.NewLineItem("cheeseburger", 500, domain.USD)
	if err != nil {
		t.Fatalf("failed to construct line item: %v", err)
	}

	first, err := sendAddLineItem(ctx, run, *li)
	if err != nil {
		t.Fatalf("first add failed: %v", err)
	}

	second, err := sendAddLineItem(ctx, run, *li)
	if err != nil {
		t.Fatalf("duplicate add failed: %v", err)
	}

	if first.LineID != second.LineID {
		t.Errorf("expected same LineID on duplicate add, got %q and %q", first.LineID, second.LineID)
	}

	if _, err := sendCloseBill(ctx, run); err != nil {
		t.Logf("cleanup close failed (non-fatal): %v", err)
	}
}

func TestBillWorkflow_CloseBill(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	run := startTestBill(t, ctx, domain.USD)

	li, err := domain.NewLineItem("cheeseburger", 500, domain.USD)
	if err != nil {
		t.Fatalf("failed to construct line item: %v", err)
	}
	if _, err := sendAddLineItem(ctx, run, *li); err != nil {
		t.Fatalf("failed to add line item: %v", err)
	}

	invoice, err := sendCloseBill(ctx, run)
	if err != nil {
		t.Fatalf("failed to close bill: %v", err)
	}

	if invoice.Status != domain.BillClosed {
		t.Errorf("Status = %v, want %v", invoice.Status, domain.BillClosed)
	}
	if invoice.TotalInvoice != 500 {
		t.Errorf("TotalInvoice = %d, want %d", invoice.TotalInvoice, 500)
	}
	if len(invoice.LineItems) != 1 {
		t.Errorf("LineItems count = %d, want %d", len(invoice.LineItems), 1)
	}

	if err := run.Get(ctx, nil); err != nil {
		t.Fatalf("workflow did not complete properly: %v", err)
	}
}

func TestBillWorkflow_GetBillState_Query(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	run := startTestBill(t, ctx, domain.USD)

	resp, err := testClient.QueryWorkflow(ctx, run.GetID(), run.GetRunID(), GetBillState)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	var result BillInvoice
	if err := resp.Get(&result); err != nil {
		t.Fatalf("failed to decode query result: %v", err)
	}

	if result.Status != domain.BillOpen {
		t.Errorf("Status = %v, want %v", result.Status, domain.BillOpen)
	}

	if _, err := sendCloseBill(ctx, run); err != nil {
		t.Logf("cleanup close failed (non-fatal): %v", err)
	}
}
