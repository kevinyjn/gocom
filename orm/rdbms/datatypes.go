package rdbms

import "xorm.io/xorm/names"

// RelationQuery for relation includes query
// examples: tables: user(id) - user_role_map(user_id, role_id) - role(id)
//   RelationTable is user_role_map
//   TargetTable is role
//   SelfRelationField is user_id
//   TargetRelationField is role_id
//   TargetPrimaryKey is (role.)id
type RelationQuery struct {
	Select              string
	RelationTable       names.TableName
	TargetTable         names.TableName
	SelfRelationField   string
	TargetRelationField string
	TargetPrimaryKey    string
}
