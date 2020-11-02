package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"

	commonkeeper "github.com/datachainlab/cross/x/atomic/common/keeper"
	simpletypes "github.com/datachainlab/cross/x/atomic/simple/types"
	"github.com/datachainlab/cross/x/core/types"
	"github.com/datachainlab/cross/x/packets"
)

const (
	TxIndexCoordinator types.TxIndex = 0
	TxIndexParticipant types.TxIndex = 1
)

type Keeper struct {
	cdc      codec.Marshaler
	storeKey sdk.StoreKey

	commonkeeper.Keeper
}

func NewKeeper(
	cdc codec.Marshaler,
	storeKey sdk.StoreKey,
	ck commonkeeper.Keeper,
) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		Keeper:   ck,
	}
}

// SendCall starts a simple commit flow
// caller is Coordinator
func (k Keeper) SendCall(
	ctx sdk.Context,
	packetSender packets.PacketSender,
	txID types.TxID,
	transactions []types.ContractTransaction,
) ([]byte, error) {
	lkr, err := types.MakeLinker(k.cdc, transactions)
	if err != nil {
		return nil, err
	}

	tx0 := transactions[TxIndexCoordinator]
	tx1 := transactions[TxIndexParticipant]

	// TODO commentout this
	// if !k.ChannelResolver().Capabilities().CrossChainCalls() && (len(tx0.Links) > 0 || len(tx1.Links) > 0) {
	// 	return nil, errors.New("this channelResolver cannot resolve cannot support the cross-chain calls feature")
	// }

	objs0, err := lkr.Resolve(tx0.Links)
	if err != nil {
		return nil, err
	}
	objs1, err := lkr.Resolve(tx1.Links)
	if err != nil {
		return nil, err
	}
	chain0, err := tx0.GetChainID(k.cdc)
	if err != nil {
		return nil, err
	}
	chain1, err := tx1.GetChainID(k.cdc)
	if err != nil {
		return nil, err
	}
	ch0, err := k.ChannelResolver().Resolve(ctx, chain0)
	if err != nil {
		return types.TxID{}, err
	}
	ch1, err := k.ChannelResolver().Resolve(ctx, chain1)
	if err != nil {
		return types.TxID{}, err
	}
	c, found := k.ChannelKeeper().GetChannel(ctx, ch1.Port, ch1.Channel)
	if !found {
		return types.TxID{}, sdkerrors.Wrap(channeltypes.ErrChannelNotFound, ch1.Channel)
	}
	if err := k.PrepareCommit(ctx, txID, TxIndexCoordinator, tx0, objs0); err != nil {
		return nil, err
	}

	// TODO define packets for simple commit
	payload := simpletypes.NewPacketDataCall(txID, types.NewContractTransactionInfo(tx1, objs1))
	if err := k.SendPacket(
		ctx,
		packetSender,
		&payload,
		ch1.Port, ch1.Channel,
		c.Counterparty.PortId, c.Counterparty.ChannelId,
		clienttypes.NewHeight(0, 0),
		0,
	); err != nil {
		return nil, err
	}
	_ = ch0

	panic("not implemented error")
}
