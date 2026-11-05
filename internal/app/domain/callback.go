package domain

import (
	"errors"
	"strconv"
	"strings"
)

type CallBack struct {
	InternalID int64  `json:"internal_id"`
	Key        string `json:"key"`
}

func (c *CallBack) String() string {
	fields := []string{
		c.Key,
		strconv.Itoa(int(c.InternalID)),
	}

	return strings.Join(fields, ":")
}

func (c *CallBack) Value(value string) error {
	fields := strings.Split(value, ":")
	if len(fields) < 2 {
		return errors.New("invalid callback value")
	}

	id, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return err
	}

	c.InternalID = id
	c.Key = fields[0]

	return nil
}
