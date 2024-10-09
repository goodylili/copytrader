package evm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math"
	"math/big"
	"net/http"
	"strings"
)

func GetETHBalance(client *ethclient.Client, address common.Address) (*big.Float, error) {
	balance, err := client.BalanceAt(context.Background(), address, nil)
	if err != nil {
		return nil, err
	}

	// Convert balance from wei to ETH
	balanceInETH := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(math.Pow10(18)))
	return balanceInETH, nil
}

func GetEthereumPrice() (int, error) {
	// Define the struct to hold the API response within the function
	type PriceResponse struct {
		Ethereum struct {
			Usd float64 `json:"usd"`
		} `json:"ethereum"`
	}

	url := "https://api.coingecko.com/api/v3/simple/price?ids=ethereum&vs_currencies=usd"

	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, errors.New("failed to fetch data from API")
	}

	var priceResp PriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
		return 0, err
	}

	// Convert the price to an integer (rounding down)
	priceInInt := int(priceResp.Ethereum.Usd)

	return priceInInt, nil
}

// CalculateTokenPriceInUSD calculates the price of the token in USD based on the amount of ETH and tokens received.
func CalculateTokenPriceInUSD(amountEth *big.Int, amountTokens *big.Int) (float64, error) {
	// Fetch the current price of Ethereum in USD
	ethPriceInUSD, err := GetEthereumPrice()
	if err != nil {
		log.Printf("Error getting Ethereum price: %v", err)
		return 0, err
	}

	// Convert the price from int to float64 for precise division
	ethPriceInUSDf := float64(ethPriceInUSD)

	// Convert the amount of ETH from wei to ETH (1 ETH = 10^18 wei)
	ethInFloat64 := new(big.Float).Quo(new(big.Float).SetInt(amountEth), big.NewFloat(1e18))

	// Convert amountTokens to float64 for division
	tokensInFloat64 := new(big.Float).SetInt(amountTokens)

	// Calculate the USD value of the ETH used in the swap
	ethValueInUSD := new(big.Float).Mul(ethInFloat64, big.NewFloat(ethPriceInUSDf))

	// Calculate the price per token in USD
	tokenPriceInUSD := new(big.Float).Quo(ethValueInUSD, tokensInFloat64)

	// Convert to float64
	tokenPriceInUSDf, _ := tokenPriceInUSD.Float64()

	return tokenPriceInUSDf, nil
}

// ERC20 ABI for the `balanceOf` function
const erc20BalanceABI = `[{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`

// GetTokenBalance retrieves the token balance of a given address for a specific ERC20 token contract.
func GetTokenBalance(client *ethclient.Client, tokenAddr, ownerAddr common.Address) (*big.Int, error) {
	// Parse the ABI
	tokenABI, err := abi.JSON(strings.NewReader(erc20BalanceABI))
	if err != nil {
		return nil, errors.New("failed to parse ABI")
	}

	// Create the call message
	data, err := tokenABI.Pack("balanceOf", ownerAddr)
	if err != nil {
		return nil, errors.New("failed to pack arguments")
	}

	callMsg := ethereum.CallMsg{
		To:   &tokenAddr,
		Data: data,
	}

	// Make the call to the token contract
	ctx := context.Background()
	result, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		return nil, errors.New("failed to call contract")
	}

	// Unpack the result into a big.Int
	var balance *big.Int
	err = tokenABI.UnpackIntoInterface(&balance, "balanceOf", result)
	if err != nil {
		return nil, errors.New("failed to unpack balance")
	}

	return balance, nil
}

// ABI for ERC20 standard functions `name` and `symbol`
const erc20ABI = `[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"}]`

// FetchTokenDetails fetches the token name and symbol for a given contract address
func FetchTokenDetails(client *ethclient.Client, contractAddress common.Address) (string, string, error) {
	// Parse the ABI
	tokenABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse ABI: %v", err)
	}

	// Create a new call context
	ctx := context.Background()

	// Fetch the token name
	callMsg := ethereum.CallMsg{To: &contractAddress, Data: tokenABI.Methods["name"].ID}
	result, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch token name: %v", err)
	}

	var tokenName string
	err = tokenABI.UnpackIntoInterface(&tokenName, "name", result)
	if err != nil {
		return "", "", fmt.Errorf("failed to unpack token name: %v", err)
	}

	// Fetch the token symbol
	callMsg.Data = tokenABI.Methods["symbol"].ID
	result, err = client.CallContract(ctx, callMsg, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch token symbol: %v", err)
	}

	var tokenSymbol string
	err = tokenABI.UnpackIntoInterface(&tokenSymbol, "symbol", result)
	if err != nil {
		return "", "", fmt.Errorf("failed to unpack token symbol: %v", err)
	}

	return tokenName, tokenSymbol, nil
}

func GetEstimatedTokensForETH(client *ethclient.Client, routerAddress, tokenAddress, WETH_ADDRESS_ common.Address, amountEth *big.Int) (*big.Int, error) {
	router, err := NewRouter(routerAddress, client)
	if err != nil {
		log.Printf("Error creating Uniswap router: %v", err)
		return nil, err
	}

	// Ethereum native token address (zero address)

	callOpts := &bind.CallOpts{
		Pending: false,
		From:    WETH_ADDRESS_,
		Context: context.Background(),
	}

	amounts, err := router.GetAmountsOut(callOpts, amountEth, []common.Address{WETH_ADDRESS_, tokenAddress})
	if err != nil {
		log.Printf("Error getting estimated tokens: %v", err)
		return nil, err
	}

	// amounts[1] will be the estimated tokens.
	log.Printf("Estimated Tokens for %s: %s", tokenAddress.Hex(), amounts[1])

	return amounts[1], nil
}

func estimateGas(client *ethclient.Client, msg ethereum.CallMsg) (uint64, error) {
	gasLimit, err := client.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, err
	}
	return gasLimit, nil
}

func CalculateMinTokens(client *ethclient.Client, routerAddress common.Address, tokenAddress, WETH_ADDRESS_ common.Address, amountEth *big.Int, slippage float64) (*big.Int, error) {
	// Estimate the number of tokens you would get for the specified ETH amount
	estimatedTokens, err := GetEstimatedTokensForETH(client, routerAddress, tokenAddress, WETH_ADDRESS_, amountEth)
	if err != nil {
		log.Printf("Error getting estimated tokens: %v", err)
		return nil, err
	}

	// Calculate the minimum number of tokens considering slippage
	slippageMultiplier := big.NewFloat(1 - slippage/100)
	minTokensFloat := new(big.Float).Mul(new(big.Float).SetInt(estimatedTokens), slippageMultiplier)

	// Convert the result from *big.Float to *big.Int
	minTokens, _ := minTokensFloat.Int(nil)

	log.Printf("Calculated Min Tokens: %s", minTokens.String())

	return minTokens, nil
}
