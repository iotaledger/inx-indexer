package indexer

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/db"
	"github.com/iotaledger/hive.go/ierrors"
	iotago "github.com/iotaledger/iota.go/v4"
)

const (
	CursorLength    = 84
	DefaultPageSize = 100
)

type LedgerUpdate struct {
	Slot     iotago.SlotIndex
	Consumed []*LedgerOutput
	Created  []*LedgerOutput
}

type LedgerOutput struct {
	OutputID iotago.OutputID
	Output   iotago.Output
	BookedAt iotago.SlotIndex
	SpentAt  iotago.SlotIndex
}

type Status struct {
	ID              uint `gorm:"primaryKey;notnull"`
	CommittedSlot   iotago.SlotIndex
	NetworkName     string
	DatabaseVersion uint32
}

type queryResult struct {
	OutputID      []byte
	Cursor        string
	CommittedSlot iotago.SlotIndex
}

type queryResults []queryResult

func (q queryResults) IDs() iotago.OutputIDs {
	outputIDs := iotago.OutputIDs{}
	for _, r := range q {
		outputIDs = append(outputIDs, iotago.OutputID(r.OutputID))
	}

	return outputIDs
}

//nolint:revive // better be explicit here
type IndexerResult struct {
	OutputIDs     iotago.OutputIDs
	CommittedSlot iotago.SlotIndex
	PageSize      uint32
	Cursor        *string
	Error         error
}

func errorResult(err error) *IndexerResult {
	return &IndexerResult{
		Error: err,
	}
}

func (i *Indexer) filteredQuery(query *gorm.DB, pageSize uint32, cursor *string) (*gorm.DB, error) {
	query = query.Select("output_id", "created_at_slot").Order("created_at_slot asc, output_id asc")
	if pageSize > 0 {
		var cursorQuery string
		//nolint:exhaustive // we have a default case.
		switch i.engine {
		case db.EngineSQLite:
			cursorQuery = "printf('%016X', created_at_slot) || hex(output_id) as cursor"
		case db.EnginePostgreSQL:
			cursorQuery = "lpad(to_hex(created_at_slot), 16, '0') || encode(output_id, 'hex') as cursor"
		default:
			i.LogFatalf("Unsupported db engine pagination queries: %s", i.engine)
		}

		// We use pageSize + 1 to load the next item to use as the cursor
		query = query.Select("output_id", "created_at_slot", cursorQuery).Limit(int(pageSize + 1))

		if cursor != nil {
			if len(*cursor) != CursorLength {
				return nil, ierrors.Errorf("Invalid cursor length: %d", len(*cursor))
			}
			//nolint:exhaustive // we have a default case.
			switch i.engine {
			case db.EngineSQLite:
				query = query.Where("cursor >= ?", strings.ToUpper(*cursor))
			case db.EnginePostgreSQL:
				query = query.Where("lpad(to_hex(created_at_slot), 16, '0') || encode(output_id, 'hex') >= ?", *cursor)
			default:
				i.LogFatalf("Unsupported db engine pagination queries: %s", i.engine)
			}
		}
	}

	return query, nil
}

func (i *Indexer) combineOutputIDFilteredQuery(query *gorm.DB, pageSize uint32, cursor *string) *IndexerResult {
	var err error
	query, err = i.filteredQuery(query, pageSize, cursor)
	if err != nil {
		return errorResult(err)
	}

	return i.resultsForQuery(query, pageSize)
}

func (i *Indexer) combineOutputIDFilteredQueries(queries []*gorm.DB, pageSize uint32, cursor *string) *IndexerResult {
	// Cast to []interface{} so that we can pass them to i.db.Raw as parameters
	filteredQueries := make([]interface{}, len(queries))
	for q, query := range queries {
		filtered, err := i.filteredQuery(query, pageSize, cursor)
		if err != nil {
			return errorResult(err)
		}
		filteredQueries[q] = filtered
	}

	unionQueryItem := "SELECT output_id, created_at_slot FROM (?) as temp;"
	if pageSize > 0 {
		unionQueryItem = "SELECT output_id, created_at_slot, cursor FROM (?) as temp;"
	}
	repeatedUnionQueryItem := strings.Split(strings.Repeat(unionQueryItem, len(queries)), ";")
	unionQuery := strings.Join(repeatedUnionQueryItem[:len(repeatedUnionQueryItem)-1], " UNION ")

	// We use pageSize + 1 to load the next item to use as the cursor
	unionQuery = fmt.Sprintf("%s ORDER BY created_at_slot asc, output_id asc LIMIT %d", unionQuery, pageSize+1)

	rawQuery := i.db.Raw(unionQuery, filteredQueries...)
	rawQuery = rawQuery.Order("created_at_slot asc, output_id asc")

	return i.resultsForQuery(rawQuery, pageSize)
}

func (i *Indexer) resultsForQuery(query *gorm.DB, pageSize uint32) *IndexerResult {
	// This combines the query with a second query that checks for the current committed_slot.
	// This way we do not need to lock anything and we know the index matches the results.
	committedSlotQuery := i.db.Model(&Status{}).Select("committed_slot")
	joinedQuery := i.db.Table("(?) as results, (?) as status", query, committedSlotQuery)

	var results queryResults

	result := joinedQuery.Find(&results)
	if err := result.Error; err != nil {
		return errorResult(err)
	}

	var committedSlot iotago.SlotIndex
	if len(results) > 0 {
		committedSlot = results[0].CommittedSlot
	} else {
		// Since we got no results for the query, return the current committedSlot
		if status, err := i.Status(); err == nil {
			committedSlot = status.CommittedSlot
		}
	}

	var nextCursor *string
	if pageSize > 0 && uint32(len(results)) > pageSize {
		lastResult := results[len(results)-1]
		results = results[:len(results)-1]
		c := strings.ToLower(lastResult.Cursor)
		nextCursor = &c
	}

	return &IndexerResult{
		OutputIDs:     results.IDs(),
		CommittedSlot: committedSlot,
		PageSize:      pageSize,
		Cursor:        nextCursor,
		Error:         nil,
	}
}
