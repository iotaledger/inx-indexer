package indexer

import (
	"context"
	"encoding/hex"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/iotaledger/hive.go/lo"
	iotago "github.com/iotaledger/iota.go/v4"
)

type multiaddress struct {
	AddressID []byte `gorm:"primaryKey;notnull"`
	Data      []byte `gorm:"notnull"`
	RefCount  int
}

func (m *multiaddress) String() string {
	return fmt.Sprintf("multiaddress => AddressID: %s", hex.EncodeToString(m.AddressID))
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
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"ref_count": gorm.Expr("ref_count + ?", multiAddr.RefCount)}),
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
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"ref_count": gorm.Expr("ref_count - ?", multiAddr.RefCount)}),
		}).Create(multiAddresses).Error; err != nil {
			return err
		}
	}

	// Delete all addresses where no references exist anymore
	return tx.Where("ref_count = ?", 0).Delete(&multiaddress{}).Error
}

func (i *Indexer) MultiAddressForReference(address *iotago.MultiAddressReference) (*iotago.MultiAddress, error) {
	var multiAddressResult multiaddress
	if err := i.db.First(&multiAddressResult, address.ID()).Error; err != nil {
		return nil, err
	}

	multiAddress := new(iotago.MultiAddress)
	if _, err := iotago.CommonSerixAPI().Decode(context.TODO(), multiAddressResult.Data, multiAddress); err != nil {
		return nil, err
	}

	return multiAddress, nil
}
