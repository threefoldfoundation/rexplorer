package main

import (
	"flag"
	"fmt"
	"html/template"
	"math/big"
	"net/http"

	"github.com/garyburd/redigo/redis"
	"github.com/threefoldfoundation/rexplorer/pkg/database"
	"github.com/threefoldfoundation/rexplorer/pkg/encoding"
	"github.com/threefoldfoundation/rexplorer/pkg/types"
	"github.com/threefoldfoundation/tfchain/pkg/config"
	"github.com/threefoldtech/rivine/pkg/client"
)

type (
	// db is the low level connection to redis
	db struct {
		conn redis.Conn
	}

	cl struct {
		db      *db
		encoder encoding.Encoder
	}

	service struct {
		cl *cl
		cc client.CurrencyConvertor
	}

	// IndexVars to render the index.html
	IndexVars struct {
		TotalCoins                                  string            `json:"total_coins"`
		LiquidCoins                                 string            `json:"liquid_coins"`
		LockedCoins                                 string            `json:"locked_coins"`
		PercentageLiquid                            string            `json:"-"`
		PercentageLocked                            string            `json:"-"`
		MinerPayouts                                string            `json:"miner_payouts"`
		TransactionFees                             string            `json:"transaction_fees"`
		FoundationFees                              string            `json:"foundation_fees"`
		TransactionCount                            uint64            `json:"transaction_count"`
		ValueTransactionCount                       uint64            `json:"value_transaction_count"`
		CoinCreationTransactionCount                uint64            `json:"coin_creation_transaction_count"`
		CoinCreatorDefinitionTransactionCount       uint64            `json:"coin_creator_definition_transaction_count"`
		ThreeBotRegistrationTransactionCount        uint64            `json:"three_bot_registration_transaction_count"`
		ThreeBotUpdateTransactionCount              uint64            `json:"three_bot_update_transaction_count"`
		BlockCreationTransactionCount               uint64            `json:"block_creation_transaction_count"`
		PercentageValueTransactions                 string            `json:"-"`
		PercentageCoinCreationTransactions          string            `json:"-"`
		PercentageCoinCreatorDefinitionTransactions string            `json:"-"`
		PercentageThreeBotRegistrationTransactions  string            `json:"-"`
		PercentageThreeBotUpdateTransactions        string            `json:"-"`
		PercentageBlockCreationTransactions         string            `json:"-"`
		BlockHeight                                 types.BlockHeight `json:"block_height"`
		Timestamp                                   types.Timestamp   `json:"timestamp"`
		CoinInputCount                              uint64            `json:"coin_input_count"`
		CoinOutputCount                             uint64            `json:"coin_output_count"`
		LiquidCoinOutputCount                       uint64            `json:"liquid_coin_output_count"`
		LockedCoinOutputCount                       uint64            `json:"locked_coin_output_count"`
		PercentageLiquidOutputs                     string            `json:"-"`
		PercentageLockedOutputs                     string            `json:"-"`
		ValueCoinOutputCount                        uint64            `json:"value_coin_output_count"`
		MinerPayoutCount                            uint64            `json:"miner_payout_count"`
		TransactionFeeCount                         uint64            `json:"transaction_fee_count"`
		FoundationFeeCount                          uint64            `json:"foundation_fee_count"`
		UniqueAddressCount                          uint64            `json:"unique_address_count"`
	}
)

var (
	dbAddress    string
	dbSlot       int
	encodingType encoding.Type
)

func init() {
	flag.StringVar(&dbAddress, "redis-addr", ":6379", "(tcp) address of the redis db")
	flag.IntVar(&dbSlot, "redis-db", 0, "slot/index of the redis db")
	flag.Var(&encodingType, "encoding", "which encdoing protocol to use, one of {json, msgp, protobuf} (default: "+encodingType.String()+")")
}

func main() {
	flag.Parse()

	encoder, err := encoding.NewEncoder(encodingType)
	if err != nil {
		panic(fmt.Sprintf("Failed to create data encoder: %v", err))
	}

	conn, err := redis.Dial("tcp", dbAddress, redis.DialDatabase(dbSlot))
	if err != nil {
		panic(fmt.Sprintf("Failed to open database: %v", err))
	}
	db := db{conn: conn}

	cl := cl{
		db:      &db,
		encoder: encoder,
	}
	defer func() {
		if err := cl.close(); err != nil {
			panic(fmt.Sprintf("Failed to close database client connection: %v", err))
		}
	}()

	cfg := config.GetBlockchainInfo()
	service := &service{
		cl: &cl,
		cc: client.NewCurrencyConvertor(config.GetCurrencyUnits(), cfg.CoinUnit),
	}

	http.HandleFunc("/", service.ShowStats)

	server := http.Server{Addr: ":8080"}
	defer func() {
		if err := server.Close(); err != nil {
			panic(fmt.Sprintf("Failed to close http server: %v", err))
		}
	}()

	if err = server.ListenAndServe(); err != nil {
		panic(fmt.Sprintf("Server failed: %v", err))
	}
}

// ShowStats shows the global stast from the redis db
func (s *service) ShowStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.cl.getGlobalStats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		return
	}

	uniqueAddresses, err := s.cl.getUniqueAddressCount()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		return
	}

	templateVars := s.indexVarsFromStats(stats, uniqueAddresses)
	err = indexTemplate.ExecuteTemplate(w, "index.html", templateVars)
	if err != nil {
		// we Can't write the header a this point as it is already set by ExecuteTemplate
		_, _ = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		return
	}
}

func (s *service) indexVarsFromStats(stats types.NetworkStats, uniqueAddresses uint64) IndexVars {
	liquidCoins := stats.Coins.Currency.Sub(stats.LockedCoins.Currency)
	i := IndexVars{}
	i.TotalCoins = s.cc.ToCoinStringWithUnit(stats.Coins.Currency)
	i.LiquidCoins = s.cc.ToCoinStringWithUnit(liquidCoins)
	i.LockedCoins = s.cc.ToCoinStringWithUnit(stats.LockedCoins.Currency)
	i.MinerPayouts = s.cc.ToCoinStringWithUnit(stats.MinerPayouts.Currency)
	i.TransactionFees = s.cc.ToCoinStringWithUnit(stats.TransactionFees.Currency)
	i.FoundationFees = s.cc.ToCoinStringWithUnit(stats.FoundationFees.Currency)
	i.TransactionCount = stats.TransactionCount
	i.ValueTransactionCount = stats.ValueTransactionCount - stats.ThreeBotRegistrationTransactionCount - stats.ThreeBotUpdateTransactionCount
	i.CoinCreationTransactionCount = stats.CoinCreationTransactionCount
	i.CoinCreatorDefinitionTransactionCount = stats.CoinCreatorDefinitionTransactionCount
	i.ThreeBotRegistrationTransactionCount = stats.ThreeBotRegistrationTransactionCount
	i.ThreeBotUpdateTransactionCount = stats.ThreeBotUpdateTransactionCount
	i.BlockCreationTransactionCount = stats.TransactionCount - stats.ValueTransactionCount - stats.CoinCreationTransactionCount - stats.CoinCreatorDefinitionTransactionCount
	i.BlockHeight = stats.BlockHeight
	i.Timestamp = stats.Timestamp
	i.CoinInputCount = stats.CoinInputCount
	i.CoinOutputCount = stats.CoinOutputCount
	i.LiquidCoinOutputCount = stats.CoinOutputCount - stats.LockedCoinOutputCount
	i.LockedCoinOutputCount = stats.LockedCoinOutputCount
	i.ValueCoinOutputCount = stats.CoinOutputCount - stats.MinerPayoutCount - stats.TransactionFeeCount
	i.MinerPayoutCount = stats.MinerPayoutCount
	i.TransactionFeeCount = stats.TransactionFeeCount
	i.FoundationFeeCount = stats.FoundationFeeCount
	i.UniqueAddressCount = uniqueAddresses

	lcpb := big.NewFloat(0).Quo(big.NewFloat(0).SetInt(liquidCoins.Big()), big.NewFloat(0).SetInt(stats.Coins.Big()))
	lcpb = lcpb.Mul(lcpb, big.NewFloat(100))
	lcp, _ := lcpb.Float64()
	i.PercentageLiquid = fmt.Sprintf("%08.5f", lcp)

	lcpb = big.NewFloat(0).Quo(big.NewFloat(0).SetInt(stats.LockedCoins.Big()), big.NewFloat(0).SetInt(stats.Coins.Big()))
	lcpb = lcpb.Mul(lcpb, big.NewFloat(100))
	lcp, _ = lcpb.Float64()
	i.PercentageLocked = fmt.Sprintf("%08.5f", lcp)

	i.PercentageValueTransactions = fmt.Sprintf("%08.5f", float64(stats.ValueTransactionCount)/float64(stats.TransactionCount)*100)
	i.PercentageCoinCreationTransactions = fmt.Sprintf("%08.5f", float64(stats.CoinCreationTransactionCount)/float64(stats.TransactionCount)*100)
	i.PercentageCoinCreatorDefinitionTransactions = fmt.Sprintf("%08.5f", float64(stats.CoinCreatorDefinitionTransactionCount)/float64(stats.TransactionCount)*100)
	i.PercentageThreeBotRegistrationTransactions = fmt.Sprintf("%08.5f", float64(stats.ThreeBotRegistrationTransactionCount)/float64(stats.TransactionCount)*100)
	i.PercentageThreeBotUpdateTransactions = fmt.Sprintf("%08.5f", float64(stats.ThreeBotUpdateTransactionCount)/float64(stats.TransactionCount)*100)
	i.PercentageBlockCreationTransactions = fmt.Sprintf("%08.5f", float64(i.BlockCreationTransactionCount)/float64(stats.TransactionCount)*100)

	i.PercentageLiquidOutputs = fmt.Sprintf("%08.5f", float64(i.LiquidCoinOutputCount)/float64(stats.CoinOutputCount)*100)
	i.PercentageLockedOutputs = fmt.Sprintf("%08.5f", float64(i.LockedCoinOutputCount)/float64(stats.CoinOutputCount)*100)

	return i
}

// getGlobalStats from the database, decoded with the used encoding type
func (c *cl) getGlobalStats() (types.NetworkStats, error) {
	var stats types.NetworkStats
	bytes, err := c.db.getGlobalStats()
	if err != nil {
		return types.NetworkStats{}, err
	}

	err = c.encoder.Unmarshal(bytes, &stats)
	return stats, err
}

// getUniqueAddressCount gets the amount of unique address which have been used in
// the chain
func (c *cl) getUniqueAddressCount() (uint64, error) {
	return c.db.getUniqueAddressCount()
}

// close the cl and underlying db
func (c *cl) close() error {
	return c.db.close()
}

// getGlobalStats from the database
func (db *db) getGlobalStats() ([]byte, error) {
	return redis.Bytes(db.conn.Do("GET", database.StatsKey))
}

// getUniqueAddressCount for the chain
func (db *db) getUniqueAddressCount() (uint64, error) {
	return redis.Uint64(db.conn.Do("SCARD", database.AddressesKey))
}

// close the connection to the database
func (db *db) close() error {
	return db.conn.Close()
}

func mustTemplate(title, templ string) *template.Template {
	p := template.New(title)
	return template.Must(p.Parse(templ))
}

var indexTemplate = mustTemplate("index.html", `
<head>
	<title>TFChain network statistics</title>
</head>
<body>
	<h2>Tfchain network has:</h2>
	<ul>
		<li>A total of {{ .TotalCoins }}</li>
		<ul>
			<li>{{ .LiquidCoins }} ({{ .PercentageLiquid }}%) liquid</li>
			<li>{{ .LockedCoins }} ({{ .PercentageLocked }}%) locked</li>
		</ul>
		<li>{{ .MinerPayouts }} is paid out as miner payout</li>
		<li>{{ .TransactionFees }} has been collected as transaction fees</li>
		<li>{{ .FoundationFees }} is paid out as foundation fees</li>
		<li>A total of {{ .TransactionCount }} transactions</li>
		<ul>
			<li>{{ .ValueTransactionCount }} ({{ .PercentageValueTransactions }}%) value transactions</li>
			<li>{{ .CoinCreationTransactionCount }} ({{ .PercentageCoinCreationTransactions }}%) coin creation transactions</li>
			<li>{{ .CoinCreatorDefinitionTransactionCount }} ({{ .PercentageCoinCreatorDefinitionTransactions }}%) coin creator definition transactions</li>
			<li>{{ .ThreeBotRegistrationTransactionCount }} ({{ .PercentageThreeBotRegistrationTransactions }}%) 3Bot registration transactions</li>
			<li>{{ .ThreeBotUpdateTransactionCount }} ({{ .PercentageThreeBotUpdateTransactions }}%) 3Bot update transactions</li>
			<li>{{ .BlockCreationTransactionCount }} ({{ .PercentageBlockCreationTransactions }}%) pure block creation transactions</li>
		</ul>
		<li>A block height of {{ .BlockHeight }}</li>
		<li>A total of {{ .ValueTransactionCount }} transactions using {{ .CoinInputCount }} coin inputs</li>
		<li>A total of {{ .CoinOutputCount }} coin outputs</li>
		<ul>
			<li>{{ .LiquidCoinOutputCount }} ({{ .PercentageLiquidOutputs }}%) liquid coin outputs</li>
			<li>{{ .LockedCoinOutputCount }} ({{ .PercentageLockedOutputs }}%) locked coin outputs</li>
		</ul>
		<br />
		<ul>
			<li>{{ .ValueCoinOutputCount }} transfer value</li>
			<li>{{ .MinerPayoutCount }} miner payouts</li>
			<li>{{ .TransactionFeeCount }} transaction fees</li>
			<li>{{ .FoundationFeeCount }} foundation fees</li>
		</ul>
		<li>A total of {{ .UniqueAddressCount }} unique addresses used</li>
</body>
`)
