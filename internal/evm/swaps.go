package evm

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
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

func SwapETHForToken(client *ethclient.Client, userWalletPrivateKey string, router, wethAddr, tokenAddress common.Address, amountInEth, minTokens *big.Int) (string, error) {
	privateKey, err := crypto.HexToECDSA(userWalletPrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(8453)) // Base network chain ID
	if err != nil {
		return "", fmt.Errorf("failed to create transaction signer: %v", err)
	}

	// Set up Uniswap router ABI
	routerABI, err := abi.JSON(strings.NewReader(ORouterABI))
	if err != nil {
		return "", fmt.Errorf("failed to parse router ABI: %v", err)
	}

	// Prepare path for token swap
	path := []common.Address{wethAddr, tokenAddress}

	// Pack the transaction data for swapExactETHForTokens
	data, err := routerABI.Pack("swapExactETHForTokens", minTokens, path, auth.From, big.NewInt(time.Now().Add(time.Minute*10).Unix()))
	if err != nil {
		return "", fmt.Errorf("failed to pack data: %v", err)
	}

	// Set up the message to estimate gas
	msg := ethereum.CallMsg{
		To:    &router,
		Value: amountInEth,
		Data:  data,
	}

	// Estimate gas
	gasLimit, err := estimateGas(client, msg)
	if err != nil {
		return "", fmt.Errorf("failed to estimate gas: %v", err)
	}

	// Get the nonce for the transaction
	nonce, err := client.PendingNonceAt(context.Background(), auth.From)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %v", err)
	}

	// Get the suggested gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %v", err)
	}

	// Create the transaction object
	tx := types.NewTransaction(nonce, router, amountInEth, gasLimit, gasPrice, data)

	// Sign the transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(8453)), privateKey) // Use chain ID for signing
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	// Return the transaction hash
	return signedTx.Hash().Hex(), nil
}

// ApproveToken approves the Uniswap router to spend token tokens on behalf of the user.
func ApproveToken(client *ethclient.Client, userWalletPrivateKey string, tokenAddress, spender common.Address, amount *big.Int) error {
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

	// Define the ERC20 token ABI (only the approve function)
	erc20ABI := `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`

	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return err
	}

	data, err := parsedABI.Pack("approve", spender, amount)
	if err != nil {
		return err
	}

	// Create the transaction
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

	// Sign the transaction
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return err
	}

	// Send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}

	fmt.Printf("Approval Transaction sent: %s\n", signedTx.Hash().Hex())

	// Wait for confirmation
	receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return errors.New("approval transaction failed")
	}

	return nil
}

// SwapTokensForETH performs a swap from token to ETH using Uniswap.
func SwapTokensForETH(client *ethclient.Client, userWalletPrivateKey string, amountIn *big.Int, tokenAddress, uniswapRouterAddress, wethAddress common.Address) error {
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

	// Load Uniswap Router ABI
	uniswapABI := `[{"inputs":[{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"uint256","name":"amountOutMin","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"swapExactTokensForETH","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"stateMutability":"nonpayable","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(uniswapABI))
	if err != nil {
		return err
	}

	amountOutMin := big.NewInt(0) // Set to 0 or calculate based on slippage
	deadline := big.NewInt(time.Now().Add(10 * time.Minute).Unix())

	path := []common.Address{
		tokenAddress,
		wethAddress,
	}

	data, err := parsedABI.Pack("swapExactTokensForETH", amountIn, amountOutMin, path, fromAddress, deadline)
	if err != nil {
		return err
	}

	// Create the transaction
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return err
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}

	tx := types.NewTransaction(nonce, uniswapRouterAddress, big.NewInt(0), 300000, gasPrice, data)

	// Sign the transaction
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)

	if err != nil {
		return err
	}

	// Send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}

	fmt.Printf("Swap Transaction sent: %s\n", signedTx.Hash().Hex())

	// Wait for confirmation
	receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return errors.New("swap transaction failed")
	}

	return nil
}
