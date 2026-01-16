package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lostboys08/ksc-go/backend/internal/database"
	"github.com/shopspring/decimal"
)

// ValidationErrorType categorizes different validation failures
type ValidationErrorType string

const (
	ErrBudgetMismatch        ValidationErrorType = "BUDGET_MISMATCH"
	ErrMonthlyAmountMismatch ValidationErrorType = "MONTHLY_AMOUNT_MISMATCH"
)

// ValidationError represents a single validation failure
type ValidationError struct {
	Type          ValidationErrorType
	ParentItemID  uuid.UUID
	ParentItemNum string
	Month         time.Time // Zero for budget validation
	ExpectedValue string    // Parent's value
	ActualValue   string    // Sum of children's values
	Difference    string    // Absolute difference
	Details       string    // Human-readable description
}

func (e ValidationError) Error() string {
	if e.Month.IsZero() {
		return fmt.Sprintf("%s: Item %s - expected budget %s, got %s (diff: %s)",
			e.Type, e.ParentItemNum, e.ExpectedValue, e.ActualValue, e.Difference)
	}
	return fmt.Sprintf("%s: Item %s, Month %s - expected %s, got %s (diff: %s)",
		e.Type, e.ParentItemNum, e.Month.Format("Jan-2006"),
		e.ExpectedValue, e.ActualValue, e.Difference)
}

// ValidationResult contains the outcome of validation
type ValidationResult struct {
	IsValid  bool
	Errors   []ValidationError
	Warnings []string
}

// DistributionResult contains the outcome of qty distribution
type DistributionResult struct {
	ItemsUpdated    int
	MonthsProcessed []time.Time
	Details         []DistributionDetail
}

// DistributionDetail describes what was distributed for a single parent
type DistributionDetail struct {
	ParentItemID      uuid.UUID
	ParentItemNum     string
	Month             time.Time
	ChildrenCount     int
	ParentPctComplete string
}

var tolerance = decimal.NewFromFloat(0.01)

// ValidateBudgetHierarchy validates that children's budgets sum to parent's budget.
// Traverses entire job hierarchy and collects all mismatches with $0.01 tolerance.
func ValidateBudgetHierarchy(ctx context.Context, q *database.Queries, jobID uuid.UUID) (*ValidationResult, error) {
	result := &ValidationResult{IsValid: true}

	// Get all parent items (items that have children)
	parentItems, err := q.GetParentItems(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("fetching parent items: %w", err)
	}

	for _, parent := range parentItems {
		children, err := q.GetDirectChildren(ctx, uuid.NullUUID{UUID: parent.ID, Valid: true})
		if err != nil {
			return nil, fmt.Errorf("fetching children for %s: %w", parent.ItemNumber, err)
		}

		if len(children) == 0 {
			continue
		}

		// Sum children's budgets
		childSum := decimal.Zero
		for _, child := range children {
			childBudget, err := decimal.NewFromString(child.Budget)
			if err != nil {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Invalid budget for item %s: %v", child.ItemNumber, err))
				continue
			}
			childSum = childSum.Add(childBudget)
		}

		parentBudget, err := decimal.NewFromString(parent.Budget)
		if err != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Invalid budget for parent %s: %v", parent.ItemNumber, err))
			continue
		}

		// Compare with tolerance
		diff := parentBudget.Sub(childSum).Abs()
		if diff.GreaterThan(tolerance) {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:          ErrBudgetMismatch,
				ParentItemID:  parent.ID,
				ParentItemNum: parent.ItemNumber,
				ExpectedValue: parentBudget.StringFixed(2),
				ActualValue:   childSum.StringFixed(2),
				Difference:    diff.StringFixed(2),
				Details:       fmt.Sprintf("Children budget sum differs from parent by $%s", diff.StringFixed(2)),
			})
		}
	}

	return result, nil
}

// ValidateMonthlyAmounts validates that children's dollar amounts sum to parent's for each month.
// Dollar amount = pay_app.qty * job_item.unit_price
func ValidateMonthlyAmounts(ctx context.Context, q *database.Queries, jobID uuid.UUID) (*ValidationResult, error) {
	result := &ValidationResult{IsValid: true}

	// Get all months with pay applications
	months, err := q.GetPayAppMonthsForJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("fetching months: %w", err)
	}

	// Get all parent items
	parentItems, err := q.GetParentItems(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("fetching parent items: %w", err)
	}

	for _, month := range months {
		for _, parent := range parentItems {
			// Get parent's pay app for this month
			parentPayApp, err := q.GetParentPayAppForMonth(ctx, database.GetParentPayAppForMonthParams{
				JobItemID:   parent.ID,
				PayAppMonth: month,
			})
			if err != nil {
				// No pay app for this parent/month combination
				continue
			}

			// Calculate parent's dollar amount
			parentQty, err := decimal.NewFromString(parentPayApp.Qty)
			if err != nil {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Invalid qty for parent %s month %s: %v",
						parent.ItemNumber, month.Format("Jan-2006"), err))
				continue
			}

			parentUnitPrice, err := decimal.NewFromString(parentPayApp.UnitPrice)
			if err != nil {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Invalid unit_price for parent %s: %v", parent.ItemNumber, err))
				continue
			}

			parentAmount := parentQty.Mul(parentUnitPrice)

			// Get children with their pay apps
			children, err := q.GetChildrenWithPayApps(ctx, database.GetChildrenWithPayAppsParams{
				ParentID:    uuid.NullUUID{UUID: parent.ID, Valid: true},
				PayAppMonth: month,
			})
			if err != nil {
				return nil, fmt.Errorf("fetching children for %s month %s: %w",
					parent.ItemNumber, month.Format("Jan-2006"), err)
			}

			// Sum children's dollar amounts
			childSum := decimal.Zero
			for _, child := range children {
				childQty, err := decimal.NewFromString(child.PayAppQty)
				if err != nil {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("Invalid qty for child %s: %v", child.ItemNumber, err))
					continue
				}

				childUnitPrice, err := decimal.NewFromString(child.UnitPrice)
				if err != nil {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("Invalid unit_price for child %s: %v", child.ItemNumber, err))
					continue
				}

				childAmount := childQty.Mul(childUnitPrice)
				childSum = childSum.Add(childAmount)
			}

			// Compare with tolerance
			diff := parentAmount.Sub(childSum).Abs()
			if diff.GreaterThan(tolerance) {
				result.IsValid = false
				result.Errors = append(result.Errors, ValidationError{
					Type:          ErrMonthlyAmountMismatch,
					ParentItemID:  parent.ID,
					ParentItemNum: parent.ItemNumber,
					Month:         month,
					ExpectedValue: parentAmount.StringFixed(2),
					ActualValue:   childSum.StringFixed(2),
					Difference:    diff.StringFixed(2),
					Details: fmt.Sprintf("Children amount sum ($%s) differs from parent ($%s) by $%s",
						childSum.StringFixed(2), parentAmount.StringFixed(2), diff.StringFixed(2)),
				})
			}
		}
	}

	return result, nil
}

// ValidateAll runs both budget and monthly amount validations.
func ValidateAll(ctx context.Context, q *database.Queries, jobID uuid.UUID) (*ValidationResult, error) {
	combined := &ValidationResult{IsValid: true}

	budgetResult, err := ValidateBudgetHierarchy(ctx, q, jobID)
	if err != nil {
		return nil, fmt.Errorf("budget validation: %w", err)
	}

	combined.Errors = append(combined.Errors, budgetResult.Errors...)
	combined.Warnings = append(combined.Warnings, budgetResult.Warnings...)
	if !budgetResult.IsValid {
		combined.IsValid = false
	}

	monthlyResult, err := ValidateMonthlyAmounts(ctx, q, jobID)
	if err != nil {
		return nil, fmt.Errorf("monthly amount validation: %w", err)
	}

	combined.Errors = append(combined.Errors, monthlyResult.Errors...)
	combined.Warnings = append(combined.Warnings, monthlyResult.Warnings...)
	if !monthlyResult.IsValid {
		combined.IsValid = false
	}

	return combined, nil
}

// DistributeParentQty distributes parent's pay application qty to children
// when parent has data but children don't for a specific month.
//
// Algorithm:
// 1. Calculate parent's percent complete: cumulative_qty_to_date / total_qty
// 2. For each child: new_qty = (child.total_qty * parent_pct) - child.previous_cumulative
// 3. Upsert child pay_applications
func DistributeParentQty(ctx context.Context, q *database.Queries, jobID uuid.UUID, targetMonth time.Time) (*DistributionResult, error) {
	result := &DistributionResult{
		MonthsProcessed: []time.Time{targetMonth},
	}

	// Get all parent items
	parentItems, err := q.GetParentItems(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("fetching parent items: %w", err)
	}

	for _, parent := range parentItems {
		// Get parent's pay app for this month
		parentPayApp, err := q.GetParentPayAppForMonth(ctx, database.GetParentPayAppForMonthParams{
			JobItemID:   parent.ID,
			PayAppMonth: targetMonth,
		})
		if err != nil {
			// No pay app for this parent/month - skip
			continue
		}

		// Check if parent has qty data
		parentQty, err := decimal.NewFromString(parentPayApp.Qty)
		if err != nil || parentQty.IsZero() {
			continue // No qty to distribute
		}

		// Get children with their pay apps
		children, err := q.GetChildrenWithPayApps(ctx, database.GetChildrenWithPayAppsParams{
			ParentID:    uuid.NullUUID{UUID: parent.ID, Valid: true},
			PayAppMonth: targetMonth,
		})
		if err != nil {
			return nil, fmt.Errorf("fetching children for %s: %w", parent.ItemNumber, err)
		}

		if len(children) == 0 {
			continue
		}

		// Check if ALL children have zero qty for this month
		anyChildHasQty := false
		for _, child := range children {
			childQty, _ := decimal.NewFromString(child.PayAppQty)
			if !childQty.IsZero() {
				anyChildHasQty = true
				break
			}
		}

		if anyChildHasQty {
			continue // Children already have data, skip distribution
		}

		// Calculate parent's percent complete from cumulative data
		parentCumulative, err := decimal.NewFromString(parentPayApp.CumulativeQty)
		if err != nil {
			continue
		}

		parentTotalQty, err := decimal.NewFromString(parentPayApp.TotalQty)
		if err != nil || parentTotalQty.IsZero() {
			continue // Cannot calculate percent complete
		}

		parentPctComplete := parentCumulative.Div(parentTotalQty)

		// Distribute to each child
		childrenUpdated := 0
		for _, child := range children {
			childTotalQty, err := decimal.NewFromString(child.TotalQty)
			if err != nil {
				continue
			}

			childPrevCumulative, err := decimal.NewFromString(child.PreviousCumulativeQty)
			if err != nil {
				childPrevCumulative = decimal.Zero
			}

			// child_new_cumulative = child.total_qty * parent_pct_complete
			childNewCumulative := childTotalQty.Mul(parentPctComplete)

			// child_month_qty = child_new_cumulative - child.previous_cumulative_qty
			childMonthQty := childNewCumulative.Sub(childPrevCumulative)

			// Ensure non-negative
			if childMonthQty.IsNegative() {
				childMonthQty = decimal.Zero
			}

			// Upsert child pay application
			err = q.UpsertPayApplication(ctx, database.UpsertPayApplicationParams{
				JobItemID:       child.ID,
				PayAppMonth:     targetMonth,
				Qty:             childMonthQty.String(),
				StoredMaterials: "0",
			})
			if err != nil {
				return nil, fmt.Errorf("upserting pay app for child %s: %w", child.ItemNumber, err)
			}

			childrenUpdated++
		}

		if childrenUpdated > 0 {
			result.ItemsUpdated += childrenUpdated
			result.Details = append(result.Details, DistributionDetail{
				ParentItemID:      parent.ID,
				ParentItemNum:     parent.ItemNumber,
				Month:             targetMonth,
				ChildrenCount:     childrenUpdated,
				ParentPctComplete: parentPctComplete.Mul(decimal.NewFromInt(100)).StringFixed(2) + "%",
			})
		}
	}

	return result, nil
}
