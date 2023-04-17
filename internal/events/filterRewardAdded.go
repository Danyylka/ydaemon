package events

import (
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/yearn/ydaemon/common/bigNumber"
	"github.com/yearn/ydaemon/common/contracts"
	"github.com/yearn/ydaemon/common/env"
	"github.com/yearn/ydaemon/common/ethereum"
	"github.com/yearn/ydaemon/common/traces"
	"github.com/yearn/ydaemon/internal/models"
)

/**************************************************************************************************
** Filter all the RewardAdded events for a the yBribe contract.
**
** Arguments:
** - chainID: the chain ID of the network we are working on
** - asyncRewardAdded: the async ptr to the map of blockNumber -> TEventRewardAdded
**
** Returns nothing as the asyncRewardAdded is updated via a pointer
**************************************************************************************************/
func filterRewardAdded(
	chainID uint64,
	asyncRewardAdded *sync.Map,
) {
	client := ethereum.GetRPC(chainID)
	contractAddress := env.YBRIBE_V3_ADDRESSES[chainID]
	currentVault, _ := contracts.NewYBribeV3(contractAddress, client)

	if log, err := currentVault.FilterRewardAdded(&bind.FilterOpts{}, nil, nil, nil); err == nil {
		for log.Next() {
			if log.Error() != nil {
				continue
			}
			asyncRewardAdded.Store(log.Event.Raw.BlockNumber, models.TEventRewardAdded{
				Amount:      bigNumber.SetInt(log.Event.Amount),
				Briber:      log.Event.Briber,
				Gauge:       log.Event.Gauge,
				RewardToken: log.Event.RewardToken,
				Timestamp:   ethereum.GetBlockTime(chainID, log.Event.Raw.BlockNumber),
				TxHash:      log.Event.Raw.TxHash,
				BlockNumber: log.Event.Raw.BlockNumber,
				TxIndex:     log.Event.Raw.TxIndex,
				LogIndex:    log.Event.Raw.Index,
			})
		}
	} else {
		traces.
			Capture(`error`, `impossible to FilterRewardAdded for YBribeV3 `+contractAddress.Hex()).
			SetEntity(`bribes`).
			SetExtra(`error`, err.Error()).
			SetTag(`chainID`, strconv.FormatUint(chainID, 10)).
			SetTag(`rpcURI`, ethereum.GetRPCURI(chainID)).
			SetTag(`bribeAddress`, contractAddress.Hex()).
			Send()
	}
}

/**********************************************************************************************
** In order to get the list, or a feed, of all the RewardAdded events, we need to filter the
** events from the blockchain and store them in a map. This function will do that.
**********************************************************************************************/
func HandleRewardsAdded(chainID uint64) map[uint64]models.TEventRewardAdded {
	if chainID != 1 {
		return make(map[uint64]models.TEventRewardAdded)
	}
	/**********************************************************************************************
	** Concurrently retrieve all first updateManagement events, waiting for the end of all
	** goroutines via the wg before continuing.
	**********************************************************************************************/
	asyncRewardAdded := sync.Map{}
	filterRewardAdded(chainID, &asyncRewardAdded)

	/**********************************************************************************************
	** Once we got all the reward added blocks, we need to extract them from the sync.Map.
	**
	** The syncMap variable is setup as follows:
	** - key: blockNumber
	** - value: TEventRewardAdded
	**
	** We need to update, for each corresponding event, the activation block to the TEventRewardAdded.
	**********************************************************************************************/
	count := 0
	rewardAddedMap := make(map[uint64]models.TEventRewardAdded)
	asyncRewardAdded.Range(func(key, value interface{}) bool {
		currentRewardAdded := value.(models.TEventRewardAdded)
		rewardAddedMap[key.(uint64)] = currentRewardAdded
		count++
		return true
	})
	return rewardAddedMap
}