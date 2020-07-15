package behaviors

import "time"

// ModifyingBehavior modifying behavior
type ModifyingBehavior struct {
	Obsoleted bool      `xorm:"'obsoleted' Bool default(0)"`
	Changes   int       `xorm:"'changes' version"`
	CreatedAt time.Time `xorm:"'created_at' DateTime created"` // If you want to save as number add Int in tag
	UpdatedAt time.Time `xorm:"'updated_at' DateTime updated"`
	CreatedBy string    `xorm:"'created_by' Varchar(50) default('')"`
	UpdatedBy string    `xorm:"'updated_by' Varchar(50) default('')"`
}
