package mysql

type CollationId uint8

const (
	DEFAULT_CHARSET                    = "utf8"
	DEFAULT_COLLATION_ID   CollationId = 33
	DEFAULT_COLLATION_NAME string      = "utf8_general_ci"
)
