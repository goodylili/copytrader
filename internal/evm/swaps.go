package evm

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const ORouterABI = `[{"constant":false,"inputs":[{"name":"amountOutMin","type":"uint256"},{"name":"path","type":"address[]"},{"name":"to","type":"address"},{"name":"deadline","type":"uint256"}],"name":"swapExactETHForTokens","outputs":[{"name":"amounts","type":"uint256[]"}],"payable":true,"stateMutability":"payable","type":"function"}]`

type ChainConfig struct {
	Name    string
	ChainID *big.Int
	RPCURL  string
}

type MultiChainRouter struct {
	Clients map[string]*ethclient.Client
	Chains  map[string]*ChainConfig
}

func NewMultiChainRouter(configs []*ChainConfig) (*MultiChainRouter, error) {
	clients := make(map[string]*ethclient.Client)
	for _, config := range configs {
		client, err := ethclient.Dial(config.RPCURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to chain %s: %v", config.Name, err)
		}
		clients[config.Name] = client
	}
	return &MultiChainRouter{
		Clients: clients,
		Chains:  make(map[string]*ChainConfig),
	}, nil
}

func (m *MultiChainRouter) SwapETHForToken(chainName, userWalletPrivateKey string, router, wethAddr, tokenAddress common.Address, amountInEth, minTokens *big.Int) (string, error) {
	client, ok := m.Clients[chainName]
	if !ok {
		return "", fmt.Errorf("unsupported chain: %s", chainName)
	}
	chainConfig, exists := m.Chains[chainName]
	if !exists {
		return "", fmt.Errorf("no configuration found for chain: %s", chainName)
	}

	privateKey, err := crypto.HexToECDSA(userWalletPrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainConfig.ChainID)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction signer: %v", err)
	}

	routerABI, err := abi.JSON(strings.NewReader(ORouterABI))
	if err != nil {
		return "", fmt.Errorf("failed to parse router ABI: %v", err)
	}

	path := []common.Address{wethAddr, tokenAddress}

	data, err := routerABI.Pack("swapExactETHForTokens", minTokens, path, auth.From, big.NewInt(time.Now().Add(time.Minute*10).Unix()))
	if err != nil {
		return "", fmt.Errorf("failed to pack data: %v", err)
	}

	msg := ethereum.CallMsg{
		To:    &router,
		Value: amountInEth,
		Data:  data,
	}

	gasLimit, err := estimateGas(client, msg)
	if err != nil {
		return "", fmt.Errorf("failed to estimate gas: %v", err)
	}

	nonce, err := client.PendingNonceAt(context.Background(), auth.From)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %v", err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %v", err)
	}

	tx := types.NewTransaction(nonce, router, amountInEth, gasLimit, gasPrice, data)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainConfig.ChainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	return signedTx.Hash().Hex(), nil
}

func (m *MultiChainRouter) ApproveToken(chainName, userWalletPrivateKey string, tokenAddress, spender common.Address, amount *big.Int) error {
	client, ok := m.Clients[chainName]
	if !ok {
		return fmt.Errorf("unsupported chain: %s", chainName)
	}
	chainConfig, exists := m.Chains[chainName]
	if !exists {
		return fmt.Errorf("no configuration found for chain: %s", chainName)
	}

	privateKey, err := crypto.HexToECDSA(userWalletPrivateKey)
	if err != nil {
		return err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	erc20ABI := `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`

	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return err
	}

	data, err := parsedABI.Pack("approve", spender, amount)
	if err != nil {
		return err
	}

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return err
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}

	msg := ethereum.CallMsg{
		From: fromAddress,
		To:   &tokenAddress,
		Data: data,
	}

	gasLimit, err := estimateGas(client, msg)
	if err != nil {
		return err
	}

	tx := types.NewTransaction(nonce, tokenAddress, big.NewInt(0), gasLimit, gasPrice, data)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainConfig.ChainID), privateKey)
	if err != nil {
		return err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}

	log.Printf("Approval Transaction sent: %s\n", signedTx.Hash().Hex())

	receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return errors.New("approval transaction failed")
	}

	return nil
}
func (m *MultiChainRouter) SwapTokensForETH(chainName, userWalletPrivateKey string, tokenAddress, uniswapRouterAddress, wethAddress common.Address, amountIn *big.Int) (string, error) {
	client, ok := m.Clients[chainName]
	if !ok {
		return "", fmt.Errorf("unsupported chain: %s", chainName)
	}

	privateKey, err := crypto.HexToECDSA(userWalletPrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Load Uniswap Router ABI
	uniswapABI := `[{"inputs":[{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"uint256","name":"amountOutMin","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"swapExactTokensForETH","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"stateMutability":"nonpayable","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(uniswapABI))
	if err != nil {
		return "", fmt.Errorf("failed to parse Uniswap ABI: %v", err)
	}

	amountOutMin := big.NewInt(0) // Set to 0 or calculate based on slippage
	deadline := big.NewInt(time.Now().Add(10 * time.Minute).Unix())

	path := []common.Address{
		tokenAddress,
		wethAddress,
	}

	data, err := parsedABI.Pack("swapExactTokensForETH", amountIn, amountOutMin, path, fromAddress, deadline)
	if err != nil {
		return "", fmt.Errorf("failed to pack data: %v", err)
	}

	// Create the transaction
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %v", err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %v", err)
	}

	tx := types.NewTransaction(nonce, uniswapRouterAddress, big.NewInt(0), 300000, gasPrice, data)

	// Sign the transaction
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get chain ID: %v", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	// Return the transaction hash directly, no waiting for receipt
	return signedTx.Hash().Hex(), nil
}
