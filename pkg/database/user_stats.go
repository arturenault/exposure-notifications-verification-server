// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package database

import (
	"time"

	"github.com/jinzhu/gorm"
)

// UserStats represents statistics related to a user in the database.
type UserStats struct {
	Date        time.Time `gorm:"date;"`
	UserID      uint      `gorm:"user_id;"`
	RealmID     uint      `gorm:"realm_id;"`
	CodesIssued uint      `gorm:"codes_issued;"`
}

// SaveUserStats saves some UserStats to the database.
// This function is provided for testing only.
func (db *Database) SaveUserStats(u *UserStats) error {
	return db.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(u).Error; err != nil {
			return err
		}
		return nil
	})
}
