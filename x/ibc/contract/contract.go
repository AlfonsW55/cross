package contract

import (
	"fmt"

	"github.com/bluele/crossccc/x/ibc/crossccc"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ crossccc.ContractHandler = (*contractHandler)(nil)

type contract struct {
	methods map[string]Method
}

func (c contract) CallMethod(ctx Context, store crossccc.Store, method string) error {
	m, ok := c.methods[method]
	if !ok {
		return fmt.Errorf("method '%v' not found", method)
	}
	return m.F(ctx, store)
}

type Method struct {
	Name string
	F    func(ctx Context, store crossccc.Store) error
}

func NewContract(methods []Method) Contract {
	mm := make(map[string]Method)
	for _, m := range methods {
		mm[m.Name] = m
	}
	return &contract{methods: mm}
}

type Contract interface {
	CallMethod(ctx Context, store crossccc.Store, method string) error
}

type contractHandler struct {
	keeper        Keeper
	routes        map[string]Contract
	stateProvider StateProvider
}

var _ crossccc.ContractHandler = (*contractHandler)(nil)

type StateProvider = func(sdk.KVStore) crossccc.State

func (h *contractHandler) Handle(ctx sdk.Context, contract []byte) (state crossccc.State, err error) {
	info, err := DecodeContractSignature(contract)
	if err != nil {
		return nil, err
	}
	st, err := h.GetState(ctx, contract)
	if err != nil {
		return nil, err
	}
	route, ok := h.routes[info.ID]
	if !ok {
		return nil, fmt.Errorf("route for '%v' not found", info.ID)
	}
	signers, ok := crossccc.SignersFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("signer is not set")
	}
	c := NewContext(signers, info.Args)

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("failed to execute a contract: %#v", e)
		}
	}()

	err = route.CallMethod(c, st, info.Method)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (h *contractHandler) GetState(ctx sdk.Context, contract []byte) (crossccc.State, error) {
	info, err := DecodeContractSignature(contract)
	if err != nil {
		return nil, err
	}
	return h.stateProvider(h.keeper.GetContractStateStore(ctx, []byte(info.ID))), nil
}

type ContractInfo struct {
	ID     string
	Method string
	Args   [][]byte
}

func NewContractInfo(id, method string, args [][]byte) ContractInfo {
	return ContractInfo{
		ID:     id,
		Method: method,
		Args:   args,
	}
}

func (ci ContractInfo) Bytes() []byte {
	bz, err := EncodeContractSignature(ci)
	if err != nil {
		panic(err)
	}
	return bz
}

func EncodeContractSignature(c ContractInfo) ([]byte, error) {
	return cdc.MarshalBinaryLengthPrefixed(c)
}

func DecodeContractSignature(bz []byte) (*ContractInfo, error) {
	var c ContractInfo
	err := cdc.UnmarshalBinaryLengthPrefixed(bz, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func NewContractHandler(k Keeper, stateProvider StateProvider) *contractHandler {
	return &contractHandler{keeper: k, routes: make(map[string]Contract), stateProvider: stateProvider}
}

func (h *contractHandler) AddRoute(id string, c Contract) {
	if _, ok := h.routes[id]; ok {
		panic("this route id already exists")
	}
	h.routes[id] = c
}

type Context interface {
	Signers() []sdk.AccAddress
	Args() [][]byte
}

type ccontext struct {
	signers []sdk.AccAddress
	args    [][]byte
}

func NewContext(signers []sdk.AccAddress, args [][]byte) Context {
	return &ccontext{signers: signers, args: args}
}

func (c ccontext) Signers() []sdk.AccAddress {
	return c.signers
}

func (c ccontext) Args() [][]byte {
	return c.args
}
