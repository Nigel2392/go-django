package page_models

type StatusFlag int64

const (
	// StatusFlagPublished is the status flag for published pages.
	StatusFlagPublished StatusFlag = 1 << iota

	// StatusFlagHidden is the status flag for hidden pages.
	StatusFlagHidden

	// StatusFlagDeleted is the status flag for deleted pages.
	StatusFlagDeleted

	// StatusflagNone is the status flag for no status.
	//
	// It is mainly used in queries to ignore the status flag in where clauses.
	StatusFlagNone StatusFlag = 0
)

func (f StatusFlag) Is(flag StatusFlag) bool {
	return f&flag == flag
}
