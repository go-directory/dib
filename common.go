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

type commonErr = common.Error

func backendError(msg ...string) (err commonErr) {
	err = common.ErrorBackend.New(msg...)
	err.SetHeader("backend:")
	return
}

func configError(msg ...string) (err commonErr) {
	err = common.ErrorConfig.New(msg...)
	err.SetHeader("config:")
	return
}
