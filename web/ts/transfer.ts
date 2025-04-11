import * as crypto from "./crypto.js";
import {Endpoint, Endpoints} from "./endpoints.js";
import * as interfaces from "./interfaces.js";

type PendingDownload = {
    id: string,
    chunks: number,
    size: number,
}

/**
 * uploadMetadata uploads file metadata to the server
 * @param metadata {object} - An object containing file metadata such as name, chunks, etc
 * @param endpoint {string} - The endpoint to use (vault or send)
 * @param callback {function(string)} - A callback returning the string ID of the file
 * @param errorCallback {function()} - An error callback for handling server errors
 */
const uploadMetadata = (
    metadata: object,
    endpoint: Endpoint,
    callback: (id: string) => void,
    errorCallback: () => void,
) => {
    let xhr = new XMLHttpRequest();
    xhr.open("POST", endpoint.path, true);
    xhr.setRequestHeader('Content-Type', 'application/json');

    xhr.onreadystatechange = () => {
        if (xhr.readyState === 4 && xhr.status === 200) {
            let response = new interfaces.MetadataUploadResponse(xhr.responseText);
            callback(response.id);
        } else if (xhr.readyState === 4 && xhr.status !== 200) {
            alert(`Error ${xhr.status}: ${xhr.responseText}`);
            errorCallback();
        }
    };

    xhr.send(JSON.stringify(metadata));
}

/**
 *
 * @param file
 * @param start
 * @param end
 * @returns {Promise<Uint8Array>}
 */
const readChunk = (file, start, end): Promise<ArrayBuffer> => {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = function(event) {
            resolve(event.target.result as ArrayBuffer);
        };

        reader.onerror = function(error) {
            reject(error);
        };

        const blob = file.slice(start, end);
        reader.readAsArrayBuffer(blob);
    });
}

/**
 * uploadChunks encrypts and uploads individual file chunks to the server until
 * the entire file has been uploaded
 * @param endpoint {string} - The string endpoint to use for uploading chunks
 * @param id {string} - The file ID returned from uploading metadata
 * @param file {File} - The file object being uploaded
 * @param key {CryptoKey} - The key to use for encrypting each file chunk
 * @param callback {function(boolean)} - A callback indicating successful upload
 * @param errorCallback {function(string)} - A callback with an error message
 */
const uploadChunks = async (
    endpoint: Endpoint,
    id: string,
    file: File,
    key: CryptoKey,
    callback: (boolean) => void,
    errorCallback: (string) => void,
) => {
    const maxConcurrentUploads = 3; // Number of workers
    const activeUploads: Set<Promise<any>> = new Set();

    let chunks = getNumChunks(file.size);
    let progressAmount = 0;
    let progressBar = document.getElementById("item-bar") as HTMLProgressElement;
    if (progressBar && chunks > 1) {
        progressBar.value = 0;
        progressBar.max = 100;
    } else if (progressBar) {
        progressBar.removeAttribute("value");
    }

    const uploadChunk = async (chunk) => {
        return new Promise(async (resolve, reject) => {
            progressAmount += 0.5;
            let progress = (progressAmount / chunks) * 100;
            if (progressBar && chunks > 1) {
                progressBar.value = progress;
            }

            let start = chunk * chunkSize;
            let end = (chunk + 1) * chunkSize;

            if (end > file.size) {
                end = file.size;
            }

            let data = await readChunk(file, start, end);
            let blob = await crypto.encryptChunk(key, new Uint8Array(data));

            sendChunk(endpoint, blob, id, chunk + 1, (response) => {
                resolve("");
                progressAmount += 0.5;
                let progress = (progressAmount / chunks) * 100;
                if (progressBar && chunks > 1) {
                    progressBar.value = progress;
                }

                if (response.length > 0) {
                    callback(true);
                } else {
                    callback(false);
                }
            }, errorMessage => {
                reject();
                errorCallback(errorMessage);
            });
        });
    }

    if (chunks === 1) {
        await uploadChunk(0);
    } else {
        for (let i = 0; i < chunks - 1; i++) {
            while (activeUploads.size >= maxConcurrentUploads) {
                await Promise.race(activeUploads);
            }

            const uploadPromise = uploadChunk(i)
                .then(() => {
                    activeUploads.delete(uploadPromise);
                })
                .catch((error) => {
                    activeUploads.delete(uploadPromise);
                    console.error(`Error uploading chunk ${i + 1}:`, error);
                });

            activeUploads.add(uploadPromise);
        }

        await Promise.all(activeUploads);
        await uploadChunk(chunks - 1);
    }
}

/**
 * sendChunk sends a chunk of encrypted file data to the server
 * @param endpoint {Endpoint} - Either shareEndpoint or vaultEndpoint
 * @param blob {Uint8Array} - The encrypted blob of file data
 * @param id {string} - The file ID returned in uploadMetadata
 * @param chunkNum {int} - The chunk #
 * @param callback {function(string)} - The server response text
 * @param errorCallback {function(string)} - The server error callback
 */
export const sendChunk = (
    endpoint: Endpoint,
    blob: Uint8Array,
    id: string,
    chunkNum: number,
    callback: (response: string) => void,
    errorCallback: (err: string) => void,
) => {
    let xhr = new XMLHttpRequest();
    let url = Endpoints.format(endpoint, id, String(chunkNum));
    xhr.open("POST", url, true);
    xhr.onreadystatechange = () => {
        if (xhr.readyState === 4 && xhr.status === 200) {
            callback(xhr.responseText);
        } else if (xhr.readyState === 4 && xhr.status !== 200) {
            errorCallback(`Error ${xhr.status}: ${xhr.responseText}`);
            throw new Error("Unable to upload chunk!");
        }
    }

    xhr.send(blob);
}

/**
 * downloadFile downloads individual file chunks from the server, decrypts them
 * using the provided key, and writes them to a file on the user's machine.
 * @param endpoint {string} - The endpoint to use for downloading file chunks
 * @param name {string} - The (previously decrypted) name of the file
 * @param download {object} - The file metadata object containing ID, size, and
 * number of chunks
 * @param key {CryptoKey} - The key to use for decrypting the file's content
 * @param callback {function(boolean)} - A function that returns true when the
 * file is finished downloading
 * @param errorCallback {function(string)} - An error callback returning the
 * error message from the server
 */
const downloadFile = (
    endpoint: Endpoint,
    name: string,
    download: PendingDownload,
    key: CryptoKey,
    callback: (success: boolean) => void,
    errorCallback: () => void,
) => {
    let writer = getFileWriter(name, download.size);

    const fetch = (chunkNum) => {
        let xhr = new XMLHttpRequest();
        let url = Endpoints.format(endpoint, download.id, chunkNum);
        xhr.open("GET", url, true);
        xhr.responseType = "blob";

        xhr.onreadystatechange = async () => {
            if (xhr.readyState === 4 && xhr.status === 200) {
                let data = new Uint8Array(await xhr.response.arrayBuffer());
                crypto.decryptChunk(key, data).then(decryptedChunk => {
                    writer.write(new Uint8Array(decryptedChunk)).then(() => {
                        if (chunkNum === download.chunks) {
                            writer.close().then(r => console.log(r));
                            callback(true);
                        } else {
                            // Fetch next chunk
                            fetch(chunkNum + 1);
                        }
                    }).catch(err => {
                        console.error(err);
                        errorCallback();
                    });
                }).catch(err => {
                    console.error(err);
                    errorCallback();
                });
            } else if (xhr.readyState === 4 && xhr.status !== 200) {
                alert(`Error ${xhr.status}: ${xhr.responseText}`);
                errorCallback();
            }
        };

        xhr.send();
    }

    // Start with first chunk
    fetch(1);
}

/**
 * Fetches a single file chunk from the given URL
 * @param url
 * @param key
 * @param successCallback
 * @param errorCallback
 */
export const fetchSingleChunk = (
    url: string,
    key: CryptoKey,
    successCallback: (Uint8Array) => void,
    errorCallback: () => void,
) => {
    fetch(url).then(response => {
        if (!response.ok) {
            errorCallback();
            return;
        }

        response.arrayBuffer().then(buf => {
            let data = new Uint8Array(buf);
            crypto.decryptChunk(key, data).then(decryptedChunk => {
                successCallback(decryptedChunk);
            }).catch(err => {
                console.error(err);
                errorCallback();
            });
        })
    })
}

/**
 * @param id {string} - The shared item ID
 * @param shareID {string} - The ID of the sharing transaction
 * @param canModify {boolean} - Whether the item can be modified
 * @param isFolder {boolean} - Whether the item is a folder or not
 */
export const changeSharedItemPerms = (id, shareID, canModify, isFolder) => {
    let endpoint = isFolder ?
        Endpoints.format(Endpoints.ShareFolder, id) :
        Endpoints.format(Endpoints.ShareFile, id);
    fetch(endpoint, {
        method: "PUT",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            id: shareID,
            itemID: id,
            canModify: canModify,
        })
    }).catch(() => {
        alert("Failed to update sharing permissions for user");
    });
}

/**
 *
 * @param id {string} - The shared item ID
 * @param shareID {string} - The ID of the sharing transaction
 * @param isFolder {boolean} - Whether the item is a folder or not
 */
export const removeUserFromShared = (id, shareID, isFolder) => {
    let endpoint = isFolder ?
        Endpoints.format(Endpoints.ShareFolder, id) :
        Endpoints.format(Endpoints.ShareFile, id);

    endpoint += `?id=${shareID}`;

    return new Promise((resolve, reject) => {
        fetch(endpoint, {method: "DELETE"}).then(() => {
            resolve(id);
        }).catch(() => {
            reject();
            alert("Failed to remove user's access to shared item");
        });
    });

}

/**
 * shareItem shares a file or folder with another YeetFile user using that recipient's
 * email or account ID.
 * @param recipient {string} - The recipient's email or account ID
 * @param rawKey {ArrayBuffer} - The decrypted file/folder key
 * @param itemID {string} - The ID of the file or folder
 * @param canModify {boolean} - Whether the recipient can modify/delete the file/folder
 * @param isFolder {boolean} - An indicator of what type of content is being shared
 */
export const shareItem = (recipient, rawKey, itemID, canModify, isFolder): Promise<interfaces.ShareInfo> => {
    let endpoint = isFolder ?
        Endpoints.format(Endpoints.ShareFolder, itemID) :
        Endpoints.format(Endpoints.ShareFile, itemID);

    return new Promise((resolve, reject) => {
        fetch(`${Endpoints.PubKey.path}?user=${recipient}`).then(async response => {
            if (!response.ok) {
                alert("Error sharing: " + await response.text());
                reject();
                return;
            }

            let recipientPublicKey = new interfaces.PubKeyResponse(
                await response.text()).publicKey;
            crypto.ingestPublicKey(recipientPublicKey, async userPubKey => {
                if (!userPubKey) {
                    alert("Error reading user's public key");
                    reject();
                    return;
                }

                let userEncItemKey = await crypto.encryptRSA(userPubKey, new Uint8Array(rawKey));
                fetch(endpoint, {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                    },
                    body: JSON.stringify({
                        user: recipient,
                        protectedKey: Array.from(userEncItemKey),
                        canModify: canModify,
                    })
                }).then(response => {
                    if (!response.ok) {
                        alert("Error sharing content with user");
                        reject();
                    } else {
                        resolve(new interfaces.ShareInfo(response));
                    }
                });
            });
        });
    });
}

/**
 *
 * @param itemID {string} - The file or folder ID
 * @param isFolder {boolean} - Whether the item is a folder
 */
export const getSharedUsers = (itemID, isFolder) => {
    let endpoint = isFolder ?
        Endpoints.format(Endpoints.ShareFolder, itemID) :
        Endpoints.format(Endpoints.ShareFile, itemID);

    return new Promise((resolve, reject) => {
        fetch(`${endpoint}`).then(response => {
            if (!response.ok) {
                alert("Error fetching shared users");
                reject();
            } else {
                resolve(response.json());
            }
        });
    });
}

const getFileWriter = (name, length) => {
    // let fileStream = streamSaver.createWriteStream(name, {
    //     size: length, // (optional filesize) Will show progress
    //     writableStrategy: undefined, // (optional)
    //     readableStrategy: undefined  // (optional)
    // });

    // StreamSaver's "mitm" technique for downloading large files only works
    // over https. If served over http, it'll default to:
    // https://jimmywarting.github.io/StreamSaver.js/mitm.html?version=2.0.0
    if (location.protocol.includes("https")) {
        window.streamSaver.mitm = "/proxy.html";
    }

    let fileStream = window.streamSaver.createWriteStream(name, {
        size: length,
    });
    return fileStream.getWriter();
}

export const uploadSendMetadata = (metadata: interfaces.UploadMetadata, callback, errorCallback) => {
    uploadMetadata(metadata, Endpoints.UploadSendFileMetadata, callback, errorCallback);
}

export const uploadVaultMetadata = (
    metadata: interfaces.VaultUpload,
    callback: (id: string) => void,
    errorCallback: () => void,
) => {
    uploadMetadata(metadata, Endpoints.UploadVaultFileMetadata, callback, errorCallback);
}

export const uploadSendChunks = async (id, file, key, callback, errorCallback) => {
    await uploadChunks(Endpoints.UploadSendFileData, id, file, key, callback, errorCallback);
}

export const uploadVaultChunks = async (id, file, key, callback, errorCallback) => {
    await uploadChunks(Endpoints.UploadVaultFileData, id, file, key, callback, errorCallback);
}

export const downloadVaultFile = (
    name: string,
    download: interfaces.VaultDownloadResponse,
    key: CryptoKey,
    callback: (success: boolean) => void,
    errorCallback: () => void,
) => {
    let pendingDownload: PendingDownload = {
        id: download.id,
        chunks: download.chunks,
        size: download.size
    }

    downloadFile(
        Endpoints.DownloadVaultFileData,
        name,
        pendingDownload,
        key,
        callback,
        errorCallback);
}

export const downloadSentFile = (name, download, key, callback, errorCallback) => {
    downloadFile(Endpoints.DownloadSendFileData, name, download, key, callback, errorCallback);
}