import * as crypto from "./crypto.js";

/**
 * Interface for vault key entries stored in IndexedDB
 */
interface VaultKeyEntry {
    id: number;
    key: Uint8Array | boolean;
}

/**
 * Interface for wordlist entries stored in IndexedDB
 */
interface WordlistEntry {
    id: number;
    list: Array<string>;
}

export class YeetFileDB {
    private readonly dbName: string;
    private readonly dbVersion: number;
    private readonly keysObjectStore: string;
    private readonly wordlistsObjectStore: string;

    private readonly privateKeyID: number;
    private readonly publicKeyID: number;
    private readonly passwordProtectedID: number;

    private readonly longWordlistID: number;
    private readonly shortWordlistID: number;

    /**
     * Converts an IndexedDB request to a Promise for easier async/await handling.
     * This helper prevents race conditions by making sure the request completes before
     * accessing its result.
     * @param request - The IDBRequest to convert
     * @returns Promise that resolves with the request result or rejects with the error
     */
    private requestToPromise<T>(request: IDBRequest<T>): Promise<T> {
        return new Promise((resolve, reject) => {
            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    }

    isPasswordProtected: (callback: (isPwProtected: boolean) => void) => void;
    insertVaultKeyPair: (
        privateKey: Uint8Array,
        publicKey: Uint8Array,
        password: string,
        callback: (success: boolean) => void,
    ) => void;
    getVaultKeyPair: (
        password: string,
        rawExport: boolean,
    ) => Promise<[CryptoKey | Uint8Array, CryptoKey | Uint8Array]>;
    removeKeys: (callback: (success: boolean) => void) => void;
    storeWordlists: (
        long: Array<string>,
        short: Array<string>,
        callback: (boolean) => void,
    ) => void;
    fetchWordlists: (
        callback: (
            success: boolean,
            long?: Array<string>,
            short?: Array<string>,
        ) => void,
    ) => void;

    constructor() {
        this.dbName = "yeetfileDB";
        this.dbVersion = 1;
        this.keysObjectStore = "keys";
        this.wordlistsObjectStore = "wordlists";

        this.privateKeyID = 1;
        this.publicKeyID = 2;
        this.passwordProtectedID = 3;

        this.longWordlistID = 1;
        this.shortWordlistID = 2;

        const request = indexedDB.open(this.dbName, this.dbVersion);

        request.onupgradeneeded = (event: IDBVersionChangeEvent) => {
            const db = (event.target as IDBOpenDBRequest)?.result;
            if (db) {
                db.createObjectStore(this.keysObjectStore, { keyPath: "id" });
                db.createObjectStore(this.wordlistsObjectStore, { keyPath: "id" });
            }
        };

        /**
         * insertKey encrypts the user's private key with a random key and stores
         * the bytes for the private and public keys in indexeddb
         * @param privateKey {Uint8Array}
         * @param publicKey {Uint8Array}
         * @param password {string}
         * @param callback {function(boolean)}
         */
        this.insertVaultKeyPair = async (
            privateKey: Uint8Array,
            publicKey: Uint8Array,
            password: string,
            callback: (arg: boolean) => void,
        ) => {
            this.removeKeys(() => { });

            let randHash = crypto.hashBlake2b(16, "JS_SESSION_KEY");

            let encKey;
            if (password.length > 0) {
                encKey = await crypto.generateArgon2Key(password, randHash);
            } else {
                encKey = await crypto.importKey(hexToBytes("JS_SESSION_KEY"));
            }

            // Replaced w/ random value on each request (needs to be cached by browser)
            let encPrivKey = await crypto.encryptChunk(encKey, privateKey);

            let request = indexedDB.open(this.dbName, this.dbVersion);
            request.onsuccess = async (event: Event) => {
                const db = (event.target as IDBOpenDBRequest)?.result;
                if (!db) {
                    callback(false);
                    return;
                }

                let transaction = db.transaction([this.keysObjectStore], "readwrite");
                let objectStore = transaction.objectStore(this.keysObjectStore);
                try {
                    let putPrivateKeyRequest = objectStore.put({
                        id: this.privateKeyID,
                        key: encPrivKey
                    });
                    putPrivateKeyRequest.onerror = (event: Event) => {
                        const error = (event.target as IDBRequest).error;
                        console.error("Error storing private key:", error);
                        callback(false);
                    };

                    let putPublicKeyRequest = objectStore.put({
                        id: this.publicKeyID,
                        key: publicKey
                    });
                    putPublicKeyRequest.onerror = (event: Event) => {
                        const error = (event.target as IDBRequest).error;
                        console.error("Error storing public key:", error);
                        callback(false);
                    };

                    let putPasswordProtectedRequest = objectStore.put({
                        id: this.passwordProtectedID,
                        key: password.length > 0
                    });
                    putPasswordProtectedRequest.onerror = (event: Event) => {
                        const error = (event.target as IDBRequest).error;
                        console.error("Error storing pw protection flag:", error);
                        callback(false);
                    };
                } catch (error) {
                    console.error("Error during put operations:", error);
                }

                transaction.onerror = (event: Event) => {
                    const error = (event.target as IDBRequest).error;
                    console.error("Error adding vault keys to IndexedDB:", error);
                    callback(false);
                };

                transaction.oncomplete = () => {
                    db.close();
                    callback(true);
                };
            }

            request.onerror = () => {
                console.error("Error opening local db");
            }
        }

        /**
         * isPasswordProtected returns whether the current vault keys were
         * encrypted with a user-provided vault password.
         * @param callback {function(boolean)}
         */
        this.isPasswordProtected = (callback) => {
            let request = indexedDB.open(this.dbName, this.dbVersion);
            request.onsuccess = async (event) => {
                const db = (event.target as IDBOpenDBRequest)?.result;
                if (!db) {
                    callback(false);
                    return;
                }

                try {
                    let transaction = db.transaction([this.keysObjectStore], "readonly");
                    let objectStore = transaction.objectStore(this.keysObjectStore);
                    const result = await this.requestToPromise<VaultKeyEntry>(objectStore.get(this.passwordProtectedID));

                    if (!result || typeof result.key !== 'boolean') {
                        console.error("Invalid or missing password protection flag in database");
                        callback(false);
                        return;
                    }

                    callback(result.key);
                } catch (error) {
                    console.error("Error checking for vault key password:", error);
                    callback(false);
                }
            }
        }

        /**
         * getVaultKey returns the vault key from the indexeddb, if it's available
         * @param password {string}
         * @param rawExport {boolean}
         */
        this.getVaultKeyPair = (
            password: string,
            rawExport: boolean,
        ): Promise<[CryptoKey | Uint8Array, CryptoKey | Uint8Array]> => {
            return new Promise((resolve, reject) => {
                let request = indexedDB.open(this.dbName, this.dbVersion);

                request.onsuccess = async (event) => {
                    const db = (event.target as IDBOpenDBRequest)?.result;
                    if (!db) {
                        reject("Unable to open db");
                        return;
                    }

                    let randHash = crypto.hashBlake2b(16, "JS_SESSION_KEY");

                    let decKey;
                    if (password.length > 0) {
                        decKey = await crypto.generateArgon2Key(password, randHash);
                    } else {
                        decKey = await crypto.importKey(hexToBytes("JS_SESSION_KEY"));
                    }

                    let transaction = db.transaction([this.keysObjectStore], "readonly");
                    let objectStore = transaction.objectStore(this.keysObjectStore);

                    try {
                        const [privateKeyResult, publicKeyResult] = await Promise.all([
                            this.requestToPromise<VaultKeyEntry>(objectStore.get(this.privateKeyID)),
                            this.requestToPromise<VaultKeyEntry>(objectStore.get(this.publicKeyID))
                        ]);

                        if (!privateKeyResult || !publicKeyResult) {
                            console.error("Vault keys not found in database");
                            reject("Vault keys not found");
                            return;
                        }

                        if (!(privateKeyResult.key instanceof Uint8Array) || !(publicKeyResult.key instanceof Uint8Array)) {
                            console.error("Invalid vault key format in database");
                            reject("Invalid vault key format");
                            return;
                        }

                        // Type assertion is safe here because we've verified the type above
                        let privateKeyBytes = privateKeyResult.key as Uint8Array;
                        const publicKeyBytes = publicKeyResult.key as Uint8Array;

                        try {
                            privateKeyBytes = await crypto.decryptChunk(decKey, privateKeyBytes);
                        } catch (error) {
                            console.error("Unable to decrypt private key:", error);
                            reject("Unable to decrypt private key");
                            return;
                        }

                        if (rawExport) {
                            resolve([privateKeyBytes, publicKeyBytes]);
                        } else {
                            crypto.ingestProtectedKey(privateKeyBytes, privateKey => {
                                crypto.ingestPublicKey(publicKeyBytes, async publicKey => {
                                    resolve([privateKey, publicKey]);
                                });
                            });
                        }
                    } catch (error) {
                        console.error("Error retrieving vault keys from IndexedDB:", error);
                        reject("Error fetching vault keys");
                    }
                }

                request.onerror = () => {
                    console.error("Error opening local db");
                    reject("Error opening local db");
                }
            });
        }

        /**
         * removeKeys removes all keys from the database and invokes the callback
         * with a boolean indicating if the removal was successful
         * @param callback {function(boolean)}
         */
        this.removeKeys = (callback) => {
            let request = indexedDB.open(this.dbName, this.dbVersion);

            request.onsuccess = (event: Event) => {
                const db = (event.target as IDBOpenDBRequest)?.result;
                if (!db) {
                    callback(false);
                    return;
                }

                let transaction = db.transaction([this.keysObjectStore], "readwrite");
                let objectStore = transaction.objectStore(this.keysObjectStore);

                let clearRequest = objectStore.clear();
                clearRequest.onsuccess = () => {
                    callback(true);
                }

                clearRequest.onerror = (event) => {
                    const error = (event.target as IDBRequest).error;
                    console.error("Error removing keys from IndexedDB:", error);
                    callback(false);
                };

                transaction.oncomplete = () => {
                    db.close();
                };
            }
        }

        /**
         * Stores the short and long wordlists in indexeddb for passphrase generation
         * @param long {Array<string>}
         * @param short {Array<string>}
         * @param callback {(boolean) => void}
         */
        this.storeWordlists = (
            long: Array<string>,
            short: Array<string>,
            callback: (boolean) => void,
        ) => {
            let request = indexedDB.open(this.dbName, this.dbVersion);
            request.onsuccess = async (event: Event) => {
                const db = (event.target as IDBOpenDBRequest)?.result;
                if (!db) {
                    callback(false);
                    return;
                }

                let transaction = db.transaction([this.wordlistsObjectStore], "readwrite");
                let objectStore = transaction.objectStore(this.wordlistsObjectStore);
                try {
                    let longWordlistRequest = objectStore.put({
                        id: this.longWordlistID,
                        list: long
                    });

                    longWordlistRequest.onerror = (event: Event) => {
                        const error = (event.target as IDBRequest).error;
                        console.error("Error storing long wordlist:", error);
                        callback(false);
                    };

                    let shortWordlistRequest = objectStore.put({
                        id: this.shortWordlistID,
                        list: short
                    });
                    shortWordlistRequest.onerror = (event: Event) => {
                        const error = (event.target as IDBRequest).error;
                        console.error("Error storing short wordlist:", error);
                        callback(false);
                    };
                } catch (error) {
                    console.error("Error during put operations:", error);
                }

                transaction.onerror = (event: Event) => {
                    const error = (event.target as IDBRequest).error;
                    console.error("Error adding wordlists to IndexedDB:", error);
                    callback(false);
                };

                transaction.oncomplete = () => {
                    db.close();
                    callback(true);
                };
            }

            request.onerror = () => {
                console.error("Error opening local db");
            }
        }

        /**
         * Checks to see if the indexeddb already contains the long and short wordlists
         * for passphrase generation.
         * @param callback {(Array<string>, Array<string>)} - The long and short wordlists
         */
        this.fetchWordlists = (
            callback: (
                success: boolean,
                long?: Array<string>,
                short?: Array<string>,
            ) => void,
        ) => {
            let request = indexedDB.open(this.dbName, this.dbVersion);
            request.onsuccess = async (event) => {
                const db = (event.target as IDBOpenDBRequest)?.result;
                if (!db) {
                    callback(false);
                    return;
                }

                try {
                    let transaction = db.transaction([this.wordlistsObjectStore], "readonly");
                    let objectStore = transaction.objectStore(this.wordlistsObjectStore);

                    const longWordlistResult = await this.requestToPromise<WordlistEntry>(objectStore.get(this.longWordlistID));
                    if (!longWordlistResult || !Array.isArray(longWordlistResult.list)) {
                        console.error("Long wordlist not found or invalid in database");
                        callback(false);
                        return;
                    }

                    const shortWordlistResult = await this.requestToPromise<WordlistEntry>(objectStore.get(this.shortWordlistID));
                    if (!shortWordlistResult || !Array.isArray(shortWordlistResult.list)) {
                        console.error("Short wordlist not found or invalid in database");
                        callback(false);
                        return;
                    }

                    callback(true, longWordlistResult.list, shortWordlistResult.list);
                } catch (error) {
                    console.error("Error checking for wordlists:", error);
                    callback(false);
                }
            }
        }
    }
}