package registries

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/yearn/ydaemon/common/env"
	"github.com/yearn/ydaemon/common/store"
	"github.com/yearn/ydaemon/internal/events"
	"github.com/yearn/ydaemon/internal/models"
)

func RegisterAllVaults(
	chainID uint64,
	start uint64,
	end *uint64,
) map[common.Address]models.TVaultsFromRegistry {
	/**********************************************************************************************
	** Our first action is to make sure we ignore the vaults we already have in our local storage,
	** which we loaded from the database.
	**********************************************************************************************/
	registriesWithLastEvent := store.ListLastNewVaultEventForRegistries(chainID)
	standardVaultList, experimentalVaultList := events.HandleNewVaults(chainID, registriesWithLastEvent, start, end)

	/**********************************************************************************************
	** Once we got all the vaults, we need to remove the duplicates. This is because some vaults
	** were deployed first as experimental vaults and then as standard vaults. In that case, we
	** keep the standard vault.
	**********************************************************************************************/
	uniqueVaultsList := make(map[common.Address]models.TVaultsFromRegistry)
	for _, v := range standardVaultList {
		uniqueVaultsList[v.VaultsAddress] = v
	}

	for _, v := range experimentalVaultList {
		if _, ok := uniqueVaultsList[v.VaultsAddress]; !ok {
			uniqueVaultsList[v.VaultsAddress] = v
		}
	}

	for _, v := range env.EXTRA_VAULTS[chainID] {
		if _, ok := uniqueVaultsList[v.VaultsAddress]; !ok {
			uniqueVaultsList[v.VaultsAddress] = v
		}
	}

	vaultsWithActivation := events.HandleUpdateManagementOneTime(chainID, uniqueVaultsList)
	for _, vault := range vaultsWithActivation {
		store.StoreNewVaultsFromRegistry(chainID, vault)
	}
	return vaultsWithActivation
}