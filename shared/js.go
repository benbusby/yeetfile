package shared

import (
	"fmt"
	"yeetfile/shared/constants"
	"yeetfile/shared/endpoints"
)

const DBFilename = "db.js"

const constsJS = `
// Auto-generated from shared/js.go. Don't edit this manually.

export const JSSessionKey = "%s";
export const IVSize = %d;
export const KeySize = %d;
export const ChunkSize = %d;
export const TotalOverhead = %d;
export const MaxTextLen = %d;
export const TextIDPrefix = "%s";
export const FileIDPrefix = "%s";
export const VerificationCodeLength = %d;
export const MaxHintLen = %d;
export const MaxPassNoteLen = %d;
export const Argon2Iter = %d;
export const Argon2Mem = %d;`

const endpointsHeadJS = `
// Auto-generated from shared/js.go. Don't edit this manually.

export type Endpoint = {
    path: string
}

export class Endpoints {`

const endpointsTailJS = `

    static format(endpoint: Endpoint, ...args: string[]): string {
        let path = endpoint.path;
        for (let arg of args) {
            path = path.replace("*", arg);
        }

        return path;
    }
}
`

const endpointEntry = `
    static %s: Endpoint = {path: "%s"};`

func GenerateSharedJS() (string, string) {
	jsConsts := fmt.Sprintf(constsJS,
		constants.JSSessionKey,
		constants.IVSize,
		constants.KeySize,
		constants.ChunkSize,
		constants.TotalOverhead,
		constants.MaxTextLen,
		constants.TextIDPrefix,
		constants.FileIDPrefix,
		constants.VerificationCodeLength,
		constants.MaxHintLen,
		constants.MaxPassNoteLen,
		constants.Argon2Iter,
		constants.Argon2Mem)

	jsEndpoints := endpointsHeadJS
	for apiEndpoint, varName := range endpoints.JSVarNameMap {
		jsEndpoints += fmt.Sprintf(endpointEntry, varName, apiEndpoint)
	}

	jsEndpoints += endpointsTailJS
	return jsConsts, jsEndpoints
}
