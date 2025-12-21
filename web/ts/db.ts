import * as crypto from "./crypto.js";

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

        const requestToPromise = <T>(request: IDBRequest<T>): Promise<T> => {
            return new Promise((resolve, reject) => {
                request.onsuccess = () => resolve(request.result);
                request.onerror = () => reject(request.error);
            });
        };

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
                    alert("Error preparing vault keys");
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
                    const result = await requestToPromise(objectStore.get(this.passwordProtectedID));
                    callback(result.key);
                } catch (error) {
                    alert("Error checking for vault key password");
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
                            requestToPromise(objectStore.get(this.privateKeyID)),
                            requestToPromise(objectStore.get(this.publicKeyID))
                        ]);

                        if (!privateKeyResult || !publicKeyResult) {
                            reject("Vault keys not found");
                            return;
                        }

                        let privateKeyBytes = privateKeyResult.key;
                        try {
                            privateKeyBytes = await crypto.decryptChunk(decKey, privateKeyBytes);
                        } catch {
                            reject("Unable to decrypt private key");
                            return;
                        }

                        if (rawExport) {
                            resolve([privateKeyBytes, publicKeyResult.key]);
                        } else {
                            crypto.ingestProtectedKey(privateKeyBytes, privateKey => {
                                let publicKey = publicKeyResult.key;
                                crypto.ingestPublicKey(publicKey, async publicKey => {
                                    resolve([privateKey, publicKey]);
                                });
                            });
                        }
                    } catch (error) {
                        console.error("Error retrieving vault keys from IndexedDB:", error);
                        alert("Error fetching vault keys");
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
                    alert("Error removing keys from IndexedDB");
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

                    const longWordlistResult = await requestToPromise(objectStore.get(this.longWordlistID));
                    if (longWordlistResult) {
                        const shortWordlistResult = await requestToPromise(objectStore.get(this.shortWordlistID));
                        callback(
                            true,
                            longWordlistResult.list as Array<string>,
                            shortWordlistResult.list as Array<string>);
                    } else {
                        callback(false);
                    }
                } catch (error) {
                    console.error("Error checking for wordlists:", error);
                    callback(false);
                }
            }
        }
    }
}