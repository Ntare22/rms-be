package users

import "time"

// ManagerBuildingAssignment scopes manager visibility to selected buildings.
type ManagerBuildingAssignment struct {
	ID             string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string    `gorm:"type:uuid;not null;uniqueIndex:uq_mgr_bld_scope,priority:1" json:"organization_id"`
	UserID         string    `gorm:"type:uuid;not null;uniqueIndex:uq_mgr_bld_scope,priority:2;index" json:"user_id"`
	BuildingID     string    `gorm:"type:uuid;not null;uniqueIndex:uq_mgr_bld_scope,priority:3;index" json:"building_id"`
	CreatedAt      time.Time `json:"created_at"`
}

func (ManagerBuildingAssignment) TableName() string { return "manager_building_assignments" }
