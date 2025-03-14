import * as constants from "./constants.js";

// @ts-ignore;
export let webcrypto;
declare const sodium: any;

const HashSize = 32;
const IVSize = 12;
let utf8Encode = new TextEncoder();
let utf8Decode = new TextDecoder();
let indexedDB: IDBFactory;

/**
 * deriveSendingKey creates a PBKDF2 key using a password as the payload, a
 * salt (or a randomly generated salt if not provided). Returns the derived key
 * and the salt.
 * @param password {string} - the password for generating the key
 * @param salt {Uint8Array} - the key salt (can be left undefined to randomly generate one)
 * @returns {Promise<[CryptoKey,Uint8Array]>}
 */
export const deriveSendingKey = async (
    password: string,
    salt: Uint8Array,
): Promise<[CryptoKey, Uint8Array]> => {
    if (!salt) {
        salt = webcrypto.getRandomValues(new Uint8Array(HashSize));
    }

    let encodedPassword = utf8Encode.encode(password);
    return [await deriveKey(encodedPassword, salt), salt];
}

/**
 * importKey imports Uint8Array key data as a CryptoKey object
 * @param keyData {Uint8Array}
 * @returns {Promise<CryptoKey>}
 */
export const importKey = async (keyData: Uint8Array): Promise<CryptoKey> => {
    return await webcrypto.subtle.importKey(
        "raw",
        keyData,
        "AES-GCM",
        true,
        ["encrypt", "decrypt"]
    )
}

/**
 * deriveKey derives a PBKDF2 key from a password and salt
 * @param password {Uint8Array} - a UTF-8 encoded password for the key
 * @param salt {Uint8Array} - the salt for the key
 * @returns {Promise<CryptoKey>}
 */
export const deriveKey = async (
    password: Uint8Array,
    salt: Uint8Array,
): Promise<CryptoKey> => {
    let keyMaterial = await webcrypto.subtle.importKey(
        "raw",
        password,
        "PBKDF2",
        false,
        ["deriveBits", "deriveKey"],
    );

    return await webcrypto.subtle.deriveKey(
        {
            name: "PBKDF2",
            salt,
            iterations: 600000,
            hash: "SHA-256",
        },
        keyMaterial,
        { name: "AES-GCM", length: 256 },
        true,
        ["encrypt", "decrypt"],
    );
}

/**
 * encryptString encrypts a string `str` using the PBKDF2 key `key` using
 * AES-GCM 256 encryption.
 * @param key {CryptoKey} - a PBKDF2 key
 * @param str {string} - the string to encrypt
 * @returns {Promise<Uint8Array>}
 */
export const encryptString = async (
    key: CryptoKey,
    str: string,
): Promise<Uint8Array> => {
    let data = utf8Encode.encode(str);
    return await encryptChunk(key, data);
}

/**
 * exportKey exports a PBKDF2 key to a Uint8Array
 * @param key {CryptoKey} - the PBKDF2 key
 * @param format {string} - the format to use when exporting the key (default "raw")
 * @returns {Promise<Uint8Array>}
 */
export const exportKey = async (
    key: CryptoKey,
    format: string,
): Promise<Uint8Array> => {
    const exported = await webcrypto.subtle.exportKey(format ? format : "raw", key);
    return new Uint8Array(exported);
}

/**
 * encryptChunk encrypts a chunk of data using the provided PBKDF2 key, and
 * returns the encrypted chunk with the initialization vector prepended to the
 * encrypted data.
 * @param key {CryptoKey} - the PBKDF2 key
 * @param data {Uint8Array} - the data to encrypt
 * @returns {Promise<Uint8Array>}
 */
export const encryptChunk = async (
    key: CryptoKey,
    data: Uint8Array,
): Promise<Uint8Array> => {
    let iv = webcrypto.getRandomValues(new Uint8Array(IVSize));
    let encrypted = await webcrypto.subtle.encrypt({ name: "AES-GCM", iv }, key, data);
    let merged = new Uint8Array(iv.length + encrypted.byteLength);
    merged.set(iv);
    merged.set(new Uint8Array(encrypted), iv.length);

    return merged;
}

/**
 * encryptRSA encrypts a Uint8Array using an RSA-OAEP public key
 * @param key {CryptoKey} - the RSA-OAEP public key
 * @param data {Uint8Array} - the data to encrypt
 * @returns {Promise<Uint8Array>}
 */
export const encryptRSA = async (
    key: CryptoKey,
    data: Uint8Array,
): Promise<Uint8Array> => {
    let encrypted = await webcrypto.subtle.encrypt({ name: "RSA-OAEP" }, key, data);
    return new Uint8Array(encrypted);
}

/**
 * decryptRSA decrypts an encrypted Uint8Array using an RSA-OAEP private key
 * @param key {CryptoKey} - the RSA-OAEP private key
 * @param data {Uint8Array} - the data to decrypt
 * @returns {Promise<Uint8Array>}
 */
export const decryptRSA = async (
    key: CryptoKey,
    data: Uint8Array,
): Promise<Uint8Array> => {
    let decrypted = await webcrypto.subtle.decrypt({ name: "RSA-OAEP" }, key, data);
    return new Uint8Array(decrypted);
}

/**
 * decryptString decrypts an encrypted string using the provided key
 * @param key {CryptoKey} - the PBKDF2 key to use for decryption
 * @param data {Uint8Array} - the encrypted string data to decrypt
 * @returns {Promise<string>}
 */
export const decryptString = async (
    key: CryptoKey,
    data: Uint8Array,
): Promise<string> => {
    let str = await decryptChunk(key, data);
    return utf8Decode.decode(str);
}

/**
 * decryptChunk decrypts a chunk of AES-GCM 256 encrypted data using
 * the provided key
 * @param key {CryptoKey} - the key to use for decryption
 * @param data {Uint8Array} - the encrypted data to decrypt
 * @returns {Promise<ArrayBuffer>}
 */
export const decryptChunk = async (
    key: CryptoKey,
    data: Uint8Array,
): Promise<Uint8Array> => {
    let iv = data.slice(0, IVSize);
    let fileData = data.slice(IVSize, data.length + 1);

    return await webcrypto.subtle.decrypt({ name: "AES-GCM", iv }, key, fileData);
}

/**
 * Generate an argon2 hash from a provided payload/password and salt.
 * @param payload
 * @param salt
 */
export const generateArgon2Key = async (
    payload: string,
    salt: Uint8Array,
): Promise<CryptoKey> => {
    await sodium.ready;

    const key = await sodium.crypto_pwhash(
        constants.KeySize,
        sodium.from_string(payload),
        salt,
        constants.Argon2Iter,
        constants.Argon2Mem * 1024 * 1024,
        sodium.crypto_pwhash_ALG_ARGON2ID13
    );

    return await importKey(key);
}

/**
 * generateUserKey creates a PBKDF2 key using user's password as the payload and
 * their identifier (email or account ID) as the salt.
 * @param identifier {string} - the user's email or account ID
 * @param password {string} - the user's password
 * @returns {Promise<CryptoKey>}
 */
export const generateUserKey = async (
    identifier: string,
    password: string,
): Promise<CryptoKey> => {
    let emailHash = hashBlake2b(16, identifier);
    return await generateArgon2Key(password, emailHash);
}

/**
 * generateLoginKeyHash generates a user's "login key" a PBKDF2 where the payload is
 * the user's user key and the salt is the user's password, and returns a SHA-256 hash
 * of that login key.
 * @param userKey {CryptoKey} - the user's user key from generateUserKey
 * @param password {string} - the user's password
 * @returns {Promise<Uint8Array>}
 */
export const generateLoginKeyHash = async (
    userKey: CryptoKey,
    password: string,
): Promise<Uint8Array> => {
    let userKeyExported = await exportKey(userKey, "raw");
    let userKeyHex = toHexString(userKeyExported);
    let pwHash = hashBlake2b(16, password);

    let loginKey = await generateArgon2Key(userKeyHex, new Uint8Array(pwHash));
    let loginKeyBytes = await exportKey(loginKey, "raw");
    let loginKeyHash = await webcrypto.subtle.digest("SHA-256", loginKeyBytes);

    return new Uint8Array(loginKeyHash);
}

/**
 * generateRandomKey uses a CSPRNG to generate a 32-byte key for encrypting and
 * decrypting folder contents in YeetFile. This is always encrypted with the user's
 * public key before being sent to the server.
 * @returns {Uint8Array}
 */
export const generateRandomKey = (): Uint8Array => {
    return webcrypto.getRandomValues(new Uint8Array(HashSize));
}

/**
 * Generates a random number between min and max
 * @param min
 * @param max
 */
export const generateRandomNumber = (min: number, max: number): number => {
    const range = max - min + 1;
    const array = new Uint32Array(1);
    let randomValue: number;

    do {
        webcrypto.getRandomValues(array);
        randomValue = array[0] / (0xFFFFFFFF + 1);
    } while (randomValue * range >= range);

    return Math.floor(randomValue * range) + min;
}

/**
 * Generates a string of cryptographically secure random characters
 * @param len
 * @param useUpper
 * @param useLower
 * @param useNumbers
 * @param useSymbols
 * @param symbols
 */
export const generateRandomString = (
    len: number,
    useUpper: boolean,
    useLower: boolean,
    useNumbers: boolean,
    useSymbols: boolean,
    symbols: string,
) => {
    const upper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";
    const lower = "abcdefghijklmnopqrstuvwxyz";
    const numbers = "1234567890";
    const defaultSymbols = "!@#$%^&*";

    const getRandomChar = (chars: string) => {
        let rand = generateRandomNumber(0, 255);
        return chars[Math.floor(rand % chars.length)];
    }

    let characters = "";
    let result = [];
    if (useUpper) {
        characters += upper;
        result.push(getRandomChar(upper));
    }

    if (useLower) {
        characters += lower;
        result.push(getRandomChar(lower));
    }

    if (useNumbers) {
        characters += numbers;
        result.push(getRandomChar(numbers));
    }

    if (useSymbols) {
        if (symbols.length > 0) {
            characters += symbols;
            result.push(getRandomChar(symbols));
        } else {
            characters += defaultSymbols;
            result.push(getRandomChar(defaultSymbols));
        }
    }

    for (let i = result.length; i < len; i++) {
        result.push(getRandomChar(characters));
    }

    result = result.sort(() => {
        let rand = generateRandomNumber(0, 255);
        return rand % 2 === 0 ? 1 : -1
    });

    return result.join("");
}

/**
 * Generates a passphrase using the provided wordlist and parameters
 * @param wordlist
 * @param numWords
 * @param separator
 * @param capitalize
 * @param useNumber
 */
export const generatePassphrase = (
    wordlist: Array<string>,
    numWords: number,
    separator: string,
    capitalize: boolean,
    useNumber: boolean,
) => {
    let passphrase = "";
    let numIdx = -1;
    if (useNumber) {
        numIdx = generateRandomNumber(0, (numWords * 2) - 1);
    }

    for (let i = 0; i < numWords; i++) {
        if (numIdx === i * 2) {
            passphrase += generateRandomNumber(0, 9);
        }

        let wordIdx = generateRandomNumber(0, wordlist.length - 1);
        let word = wordlist[wordIdx];

        if (capitalize) {
            word = word.substring(0, 1).toUpperCase() + word.substring(1).toLowerCase();
        }

        passphrase += word;

        if (numIdx === (i * 2) + 1) {
            passphrase += generateRandomNumber(0, 9);
        }

        if (i < numWords - 1) {
            passphrase += separator;
        }
    }

    return passphrase;
}

/**
 * ingestPublicKey takes the raw base64 of the user's public key and
 * converts them into a CryptoKey object that can be used for encryption.
 * @param publicKey {Uint8Array}
 * @param callback {function(CryptoKey)}
 */
export const ingestPublicKey = (
    publicKey: Uint8Array,
    callback: (arg: CryptoKey) => void,
) => {
    webcrypto.subtle.importKey(
        "spki",
        publicKey,
        {
            name: "RSA-OAEP",
            hash: { name: "SHA-256" }
        },
        false,
        ["encrypt"]
    ).catch((error: Error) => {
        console.error("Error re-importing vault key:", error);
    }).then((key: CryptoKey) => {
        callback(key);
    });
}

/**
 * ingestProtectedKey creates a non-exportable CryptoKey object of the
 * user's private key
 * @param protectedKey {Uint8Array}
 * @param callback {function(CryptoKey)}
 */
export const ingestProtectedKey = (
    protectedKey: Uint8Array,
    callback: (arg: CryptoKey) => void,
) => {
    // Import the key as non-exportable
    webcrypto.subtle.importKey(
        "pkcs8",
        protectedKey,
        {
            name: "RSA-OAEP",
            hash: { name: "SHA-256" }
        },
        false,
        ["decrypt"])
        .catch((error: Error) => {
            console.error("Error re-importing vault key:", error);
        })
        .then((key: CryptoKey) => {
            callback(key);
        });
}

/**
 * generateKeyPair generates RSA-OAEP public + private keys. The private key
 * is used for encrypting/decrypting the user's root folder, as well as folder
 * keys that are shared with the user. The public key is used by other users to
 * share folders.
 *
 * Note that the generated key pair is marked as "extractable", since the private
 * key must be further encrypted by the user key before being sent to the server.
 *
 * @returns {Promise<CryptoKeyPair>}
 */
export const generateKeyPair = async (): Promise<CryptoKeyPair> => {
    return await webcrypto.subtle.generateKey(
        {
            name: "RSA-OAEP",
            modulusLength: 2048,
            publicExponent: new Uint8Array([0x01, 0x00, 0x01]), // 65537
            hash: { name: "SHA-256" }
        }, true, ["encrypt", "decrypt"]
    );
}

/**
 * unwindKeys unwinds an ordered key sequence for a file or folder. The sequence
 * contains every parent folder's protected key, which is used to decrypt each
 * child's key.
 * @param privateKey {CryptoKey}
 * @param keySequence {Uint8Array[]}
 */
export const unwindKeys = async (privateKey: CryptoKey, keySequence: Uint8Array[]) => {
    let parentKey;
    for (let i = 0; i < keySequence.length; i++) {
        if (!parentKey) {
            let protectedKey = keySequence[i];
            parentKey = await decryptRSA(privateKey, protectedKey);
            continue;
        }

        let parentKeyImport = await importKey(parentKey);
        let protectedKey = keySequence[i];
        parentKey = await decryptChunk(parentKeyImport, protectedKey);
    }

    return await importKey(parentKey);
}

export const hashBlake2b = (len: number, input: string): Uint8Array => {
    if (len > 32) {
        len = 32;
    } else if (len < 0) {
        len = 1;
    }

    let utf8Input = utf8Encode.encode(input);
    return sodium.crypto_generichash(32, utf8Input).subarray(0, len);

}

if (typeof window !== "undefined") {
    // Enforce browser variables
    webcrypto = window.crypto;
    // @ts-ignore
    indexedDB = window.indexedDB || window.mozIndexedDB || window.webkitIndexedDB || window.msIndexedDB || window.shimIndexedDB
} else {
    // @ts-ignore
    webcrypto = await import("crypto");
}
