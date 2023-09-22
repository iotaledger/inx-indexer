package indexer

import (
	"context"
	"encoding/hex"
	"fmt"

	iotago "github.com/iotaledger/iota.go/v4"
)

type multiaddress struct {
	AddressID []byte `gorm:"primaryKey;notnull"`
	Data      []byte `gorm:"notnull"`
}

func (m *multiaddress) String() string {
	return fmt.Sprintf("multiaddress => AddressID: %s", hex.EncodeToString(m.AddressID))
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
	// We also de-dup the addresses here to avoid double insertions.
	alreadyKnownAddresses := make(map[string]struct{}, 0)
	multiAddresses := make([]*multiaddress, 0)
	for _, address := range addresses {
		if multiAddress := multiAddressFromAddress(address); multiAddress != nil {
			if _, alreadyKnown := alreadyKnownAddresses[multiAddress.Key()]; !alreadyKnown {
				addrData, err := iotago.CommonSerixAPI().Encode(context.TODO(), multiAddress)
				if err != nil {
					return nil, err
				}

				multiAddresses = append(multiAddresses, &multiaddress{
					AddressID: multiAddress.ID(),
					Data:      addrData,
				})

				alreadyKnownAddresses[multiAddress.Key()] = struct{}{}
			}
		}
	}

	return multiAddresses, nil
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
