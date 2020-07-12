package behaviors

import "time"

// ModifyingBehavior modifying behavior
type ModifyingBehavior struct {
	Obsoleted bool      `xorm:"'obsoleted' BOOL default(0)"`
	Changes   int       `xorm:"'changes' version"`
	CreatedAt time.Time `xorm:"'created_at' created"` // If you want to save as number add Int in tag
	UpdatedAt time.Time `xorm:"'updated_at' updated"`
	CreatedBy string    `xorm:"'created_by' VARCHAR(50) default('')"`
	UpdatedBy string    `xorm:"'updated_by' VARCHAR(50) default('')"`
}
