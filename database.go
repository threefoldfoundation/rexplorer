package main

import (
	"encoding/json"
	"log"

	"github.com/rivine/rivine/modules"
	"github.com/rivine/rivine/types"
)

type Database interface {
	GetExplorerState() (ExplorerState, error)
	SetExplorerState(state ExplorerState) error

	GetNetworkStats() (NetworkStats, error)
	SetNetworkStats(stats NetworkStats) error

	AddMinerPayout(uh types.UnlockHash, amount types.Currency) error
	RemoveMinerPayout(uh types.UnlockHash, amount types.Currency) error

	AddCoinOutput(id types.CoinOutputID, co types.CoinOutput) error
	RemoveCoinOutput(id types.CoinOutputID) error
}

type (
	RedisDatabase struct{}
)

var (
	_ Database = (*RedisDatabase)(nil)
)

func NewRedisDatabase(address string, db int) (*RedisDatabase, error) {
	// TODO
	return &RedisDatabase{}, nil
}

func (rdb *RedisDatabase) GetExplorerState() (ExplorerState, error) {
	// TODO: load from redis
	return ExplorerState{
		CurrentChangeID: modules.ConsensusChangeBeginning,
	}, nil
}

func (rdb *RedisDatabase) SetExplorerState(state ExplorerState) error {
	// TODO: store to redis
	b, _ := json.Marshal(state)
	log.Println(string(b))
	return nil
}

func (rdb *RedisDatabase) GetNetworkStats() (NetworkStats, error) {
	// TODO: load from redis
	return NetworkStats{}, nil
}

func (rdb *RedisDatabase) SetNetworkStats(stats NetworkStats) error {
	// TODO: store to redis
	b, _ := json.Marshal(stats)
	log.Println(string(b))
	return nil
}

func (rdb *RedisDatabase) AddMinerPayout(uh types.UnlockHash, amount types.Currency) error {
	// TODO: store to redis
	log.Printf("add miner payout %s for %s", amount.String(), uh.String())
	return nil
}

func (rdb *RedisDatabase) RemoveMinerPayout(uh types.UnlockHash, amount types.Currency) error {
	// TODO: store to redis
	log.Printf("remove miner payout %s for %s", amount.String(), uh.String())
	return nil
}

func (rdb *RedisDatabase) AddCoinOutput(id types.CoinOutputID, co types.CoinOutput) error {
	// TODO: store to redis
	log.Printf("add coin output %x for %s", id, co.Condition.UnlockHash().String())
	return nil
}

func (rdb *RedisDatabase) RemoveCoinOutput(id types.CoinOutputID) error {
	// TODO: store to redis
	log.Printf("remove coin output %x", id)
	return nil
}
