package main

import (
	"fmt"
	"github.com/tkrajina/typescriptify-golang-structs/typescriptify"
	"log"
	"os"
	"yeetfile/shared"
)

const jsConstsFile = "constants.ts"
const structsFile = "interfaces.ts"
const jsEndpointsFile = "endpoints.ts"

func write(path, contents string) {
	if err := os.WriteFile(path, []byte(contents), 0666); err != nil {
		log.Fatal(err)
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Must specify output directory")
	}

	outDir := os.Args[1]
	consts, endpoints := shared.GenerateSharedJS()
	if _, err := os.Stat(outDir); err != nil {
		log.Fatal(err)
	}

	constsOut := fmt.Sprintf("%s/%s", outDir, jsConstsFile)
	structsOut := fmt.Sprintf("%s/%s", outDir, structsFile)
	endpointsOut := fmt.Sprintf("%s/%s", outDir, jsEndpointsFile)

	write(constsOut, consts)
	write(endpointsOut, endpoints)

	fmt.Printf("TypeScript constants written to: %s\n", constsOut)
	fmt.Printf("TypeScript endpoints written to: %s\n", endpointsOut)

	// Disable output for typescriptify (too verbose w/ no way to disable)
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	old := os.Stdout
	os.Stdout = f

	converter := typescriptify.New().
		Add(shared.UploadMetadata{}).
		Add(shared.VaultUpload{}).
		Add(shared.ModifyVaultItem{}).
		Add(shared.MetadataUploadResponse{}).
		Add(shared.NewFolderResponse{}).
		Add(shared.VaultItem{}).
		Add(shared.VaultItemInfo{}).
		Add(shared.NewVaultFolder{}).
		Add(shared.NewPublicVaultFolder{}).
		Add(shared.VaultFolder{}).
		Add(shared.VaultFolderResponse{}).
		Add(shared.VaultDownloadResponse{}).
		Add(shared.TextUpload{}).
		Add(shared.DownloadResponse{}).
		Add(shared.Signup{}).
		Add(shared.SignupResponse{}).
		Add(shared.VerifyAccount{}).
		Add(shared.Login{}).
		Add(shared.LoginResponse{}).
		Add(shared.SessionInfo{}).
		Add(shared.ForgotPassword{}).
		Add(shared.ResetPassword{}).
		Add(shared.PubKeyResponse{}).
		Add(shared.ShareItemRequest{}).
		Add(shared.NewSharedItem{}).
		Add(shared.FileOwnershipInfo{}).
		Add(shared.FolderOwnershipInfo{}).
		Add(shared.ShareInfo{}).
		Add(shared.ShareEdit{}).
		Add(shared.DeleteResponse{}).
		Add(shared.DeleteAccount{}).
		Add(shared.ChangePassword{}).
		Add(shared.ProtectedKeyResponse{}).
		Add(shared.VerifyEmail{}).
		Add(shared.ChangePasswordHint{}).
		Add(shared.StartEmailChangeResponse{}).
		Add(shared.ChangeEmail{}).
		Add(shared.NewTOTP{}).
		Add(shared.SetTOTP{}).
		Add(shared.SetTOTPResponse{}).
		Add(shared.ItemIndex{}).
		Add(shared.AdminUserInfoResponse{}).
		Add(shared.AdminFileInfoResponse{}).
		Add(shared.ServerInfo{})

	converter.WithBackupDir("")
	err = converter.ConvertToFile(structsOut)
	if err != nil {
		panic(err.Error())
	}

	os.Stdout = old
	fmt.Printf("TypeScript interfaces written to: %s\n", structsOut)
}
