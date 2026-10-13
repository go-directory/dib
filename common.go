package dib

import (
	"github.com/go-directory/common"
)

type BitValue = common.BitValue

func new8BitValue() BitValue {
	return common.NewBitValue(common.Uint8)
}

func new32BitValue() BitValue {
	return common.NewBitValue(common.Uint32)
}

func backendError(msg ...string) error {
	m := append([]string{"Backend error: "}, msg...)
	return common.ErrorBackend.New(m...)
}

func configError(msg ...string) error {
	m := append([]string{"Config error: "}, msg...)
	return common.ErrorConfig.New(m...)
}
