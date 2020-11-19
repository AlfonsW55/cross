package cli

import (
	"encoding/hex"
	"io/ioutil"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/types/time"

	"github.com/datachainlab/cross/x/core/types"
)

// NewInitiateTxCmd returns the command to create a NewMsgInitiateTx transaction
func NewInitiateTxCmd() *cobra.Command {
	const (
		flagContractTransactions = "contract-txs"
	)

	cmd := &cobra.Command{
		Use:   "initiate-tx",
		Short: "Create a NewMsgInitiateTx transaction",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			clientCtx, err := client.ReadTxCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			sender := types.AccountIDFromAccAddress(clientCtx.GetFromAddress())
			ctxs, err := readContractTransactions(clientCtx.JSONMarshaler, viper.GetStringSlice(flagContractTransactions))
			if err != nil {
				return err
			}
			msg := types.NewMsgInitiateTx(
				sender,
				clientCtx.ChainID,
				0,
				types.COMMIT_PROTOCOL_SIMPLE,
				ctxs,
				clienttypes.ZeroHeight(),
				uint64(time.Now().Unix())+1000,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().StringSlice(flagContractTransactions, nil, "A file path to includes a contract transaction")
	cmd.MarkFlagRequired(flagContractTransactions)

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func readContractTransactions(m codec.JSONMarshaler, pathList []string) ([]types.ContractTransaction, error) {
	var cTxs []types.ContractTransaction
	for _, path := range pathList {
		bz, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var cTx types.ContractTransaction
		if err := m.UnmarshalJSON(bz, &cTx); err != nil {
			return nil, err
		}
		cTxs = append(cTxs, cTx)
	}
	return cTxs, nil
}

func NewIBCSignTxCmd() *cobra.Command {
	const (
		flagTxID                  = "tx-id"
		flagInitiatorChainChannel = "initiator-chain-channel"
	)

	cmd := &cobra.Command{
		Use:   "ibc-signtx",
		Short: "Sign the cross-chain transaction on other chain via the chain",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			clientCtx, err := client.ReadTxCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			anyXCC, err := resolveAnyXCCFromChannelString(
				channeltypes.NewQueryClient(clientCtx),
				viper.GetString(flagInitiatorChainChannel),
			)
			if err != nil {
				return err
			}
			signer := types.AccountIDFromAccAddress(clientCtx.FromAddress)
			txID, err := hex.DecodeString(viper.GetString(flagTxID))
			if err != nil {
				return err
			}
			msg := types.NewMsgIBCSignTx(
				anyXCC,
				txID,
				[]types.AccountID{signer},
				clienttypes.ZeroHeight(),
				uint64(time.Now().Unix())+1000,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(flagTxID, "", "hex encoding of the TxID")
	cmd.Flags().String(flagInitiatorChainChannel, "", "channel info: '<channelID>:<portID>'")
	cmd.MarkFlagRequired(flagTxID)
	cmd.MarkFlagRequired(flagInitiatorChainChannel)

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
