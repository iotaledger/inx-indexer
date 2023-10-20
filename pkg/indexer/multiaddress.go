package indexer

import (
	"context"
	"encoding/hex"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/lo"
	iotago "github.com/iotaledger/iota.go/v4"
)

type multiaddress struct {
	AddressID []byte `gorm:"primaryKey;notnull"`
	Data      []byte `gorm:"notnull"`
	RefCount  int
}

var (
	ErrMultiAddressNotFound = ierrors.Errorf("multi address not found")
)

func (m *multiaddress) String() string {
	return fmt.Sprintf("multiaddress => AddressID: %s", hex.EncodeToString(m.AddressID))
}

func (m *multiaddress) primaryKeyRow() string {
	return "address_id"
}

func (m *multiaddress) refCountDelta() int {
	return m.RefCount
}

func multiAddressesForAddresses(addresses ...iotago.Address) ([]*multiaddress, error) {
	multiAddressFromAddress := func(address iotago.Address) *iotago.MultiAddress {
		if multi, isMulti := address.(*iotago.MultiAddress); isMulti {
			return multi
		}

		if restricted, isRestricted := address.(*iotago.RestrictedAddress); isRestricted {
			if innerMulti, innerIsMulti := restricted.Address.(*iotago.MultiAddress); innerIsMulti {
				return innerMulti
			}
		}

		return nil
	}

	// Store all passed addresses if they are or contain a MultiAddress.
	// We increase the counter for duplicate addresses
	multiAddresses := make(map[string]*multiaddress, 0)
	for _, address := range addresses {
		if multiAddress := multiAddressFromAddress(address); multiAddress != nil {
			if multiAddr, alreadyKnown := multiAddresses[multiAddress.Key()]; !alreadyKnown {
				addrData, err := iotago.CommonSerixAPI().Encode(context.TODO(), multiAddress)
				if err != nil {
					return nil, err
				}

				multiAddresses[multiAddress.Key()] = &multiaddress{
					AddressID: multiAddress.ID(),
					Data:      addrData,
					RefCount:  1,
				}
			} else {
				multiAddr.RefCount++
			}
		}
	}

	return lo.Values(multiAddresses), nil
}

func insertMultiAddressesFromAddresses(tx *gorm.DB, addresses []iotago.Address) error {
	multiAddresses, err := multiAddressesForAddresses(addresses...)
	if err != nil {
		return err
	}

	for _, multiAddr := range multiAddresses {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: multiAddr.primaryKeyRow()}},
			DoUpdates: clause.Assignments(map[string]interface{}{"ref_count": gorm.Expr("ref_count + ?", multiAddr.refCountDelta())}),
		}).Create(multiAddresses).Error; err != nil {
			return err
		}
	}

	return nil
}

func deleteMultiAddressesFromAddresses(tx *gorm.DB, addresses []iotago.Address) error {
	multiAddresses, err := multiAddressesForAddresses(addresses...)
	if err != nil {
		return err
	}

	for _, multiAddr := range multiAddresses {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: multiAddr.primaryKeyRow()}},
			DoUpdates: clause.Assignments(map[string]interface{}{"ref_count": gorm.Expr("ref_count - ?", multiAddr.refCountDelta())}),
		}).Create(multiAddresses).Error; err != nil {
			return err
		}
	}

	// Delete all addresses where no references exist anymore
	return tx.Where("ref_count = ?", 0).Delete(&multiaddress{}).Error
}

func (i *Indexer) MultiAddressForReference(address *iotago.MultiAddressReference) (*iotago.MultiAddress, error) {
	var multiAddressResult multiaddress
	if err := i.db.Model(&multiaddress{}).Where("address_id = ?", address.MultiAddressID).Find(&multiAddressResult).Error; err != nil {
		return nil, err
	}

	if multiAddressResult.AddressID == nil {
		return nil, ErrMultiAddressNotFound
	}

	multiAddress := new(iotago.MultiAddress)
	if _, err := iotago.CommonSerixAPI().Decode(context.TODO(), multiAddressResult.Data, multiAddress); err != nil {
		return nil, err
	}

	return multiAddress, nil
}
