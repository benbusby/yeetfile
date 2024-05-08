import * as crypto from "./crypto.js";

const sendEndpoint = "/send";
const vaultEndpoint = "/api/vault"

/**
 * uploadMetadata uploads file metadata to the server
 * @param metadata {object} - An object containing file metadata such as name, chunks, etc
 * @param endpoint {string} - The endpoint (either shareEndpoint or vaultEndpoint) to use
 * @param callback {function(string)} - A callback returning the string ID of the file
 * @param errorCallback {function()} - An error callback for handling server errors
 */
const uploadMetadata = (metadata, endpoint, callback, errorCallback) => {
    let xhr = new XMLHttpRequest();
    xhr.open("POST", endpoint + "/u", true);
    xhr.setRequestHeader('Content-Type', 'application/json');

    xhr.onreadystatechange = () => {
        if (xhr.readyState === 4 && xhr.status === 200) {
            let response = JSON.parse(xhr.response);
            callback(response.id);
        } else if (xhr.readyState === 4 && xhr.status !== 200) {
            alert(`Error ${xhr.status}: ${xhr.responseText}`);
        }
    };

    xhr.send(JSON.stringify(metadata));
}

/**
 *
 * @param file
 * @param start
 * @param end
 * @returns {Promise<ArrayBuffer>}
 */
const readChunk = (file, start, end) => {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = function(event) {
            resolve(event.target.result);
        };

        reader.onerror = function(error) {
            reject(error);
        };

        const blob = file.slice(start, end);
        reader.readAsArrayBuffer(blob);
    });
}

/**
 * uploadChunks encrypts and uploads individual file chunks to the server until the entire file
 * has been uploaded
 * @param endpoint {string} - The string endpoint to upload chunks to (either sendEndpoint or vaultEndpoint)
 * @param id {string} - The file ID returned during the initial file metadata creation step beforehand
 * @param file {File} - The file object being uploaded
 * @param key {CryptoKey} - The key to use for encrypting each file chunk
 * @param callback {function(boolean)} - A callback returning true when the file is finished uploading
 * @param errorCallback {function(string)} - An error callback for any errors with uploading the file
 */
const uploadChunks = async (endpoint, id, file, key, callback, errorCallback) => {
    let chunks = getNumChunks(file.size);
    let progressBar = document.getElementById("item-bar");
    if (progressBar && chunks > 1) {
        progressBar.value = 0;
        progressBar.max = 100;
    } else if (progressBar) {
        progressBar.removeAttribute("value");
    }

    const uploadChunk = async (chunk) => {
        let progress = ((chunk + 1) / chunks) * 100;
        console.log("Uploading: " + progress + "%");
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
            if (response.length > 0) {
                callback(true);
            } else {
                uploadChunk(chunk + 1);
            }
        }, errorMessage => {
            errorCallback(errorMessage);
        });
    }

    await uploadChunk(0);
}

/**
 * sendChunk sends a chunk of encrypted file data to the server
 * @param endpoint {string} - Either shareEndpoint or vaultEndpoint
 * @param blob {Uint8Array} - The encrypted blob of file data
 * @param id {string} - The file ID returned in uploadMetadata
 * @param chunkNum {int} - The chunk #
 * @param callback {function(string)} - The server response text
 * @param errorCallback {function(string)} - The server error callback
 */
export const sendChunk = (endpoint, blob, id, chunkNum, callback, errorCallback) => {
    let xhr = new XMLHttpRequest();
    xhr.open("POST", `${endpoint}/u/${id}/${chunkNum}`, true);
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
 * downloadFile downloads individual file chunks from the server, decrypts them using
 * the provided key, and writes them to a file on the user's machine.
 * @param endpoint {string} - The endpoint to use for downloading file chunks
 * @param name {string} - The (previously decrypted) name of the file
 * @param download {object} - The file metadata object containing ID, size, and number of chunks
 * @param key {CryptoKey} - The key to use for decrypting the file's content
 * @param callback {function(boolean)} - A function that returns true when the file is finished downloading
 * @param errorCallback {function(string)} - An error callback returning the error message from the server
 */
const downloadFile = (endpoint, name, download, key, callback, errorCallback) => {
    let writer = getFileWriter(name, download.size);

    const fetch = (chunkNum) => {
        let xhr = new XMLHttpRequest();
        xhr.open("GET", `${endpoint}/d/${download.id}/${chunkNum}`, true);
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
                    });
                }).catch(err => {
                    console.error(err);
                });

            } else if (xhr.readyState === 4 && xhr.status !== 200) {
                alert(`Error ${xhr.status}: ${xhr.responseText}`);
                callback(false);
            }
        };

        xhr.send();
    }

    // Start with first chunk
    fetch(1);
}

/**
 * @param id {string} - The shared item ID
 * @param shareID {string} - The ID of the sharing transaction
 * @param canModify {boolean} - Whether the item can be modified
 * @param isFolder {boolean} - Whether the item is a folder or not
 */
export const changeSharedItemPerms = (id, shareID, canModify, isFolder) => {
    let endpoint = isFolder ? `/api/share/folder/${id}` : `/api/share/file/${id}`;
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
        `/api/share/folder/${id}?id=${shareID}` :
        `/api/share/file/${id}?id=${shareID}`;

    return new Promise((resolve, reject) => {
        fetch(endpoint, {method: "DELETE"}).then(() => {
            resolve();
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
export const shareItem = (recipient, rawKey, itemID, canModify, isFolder) => {
    let endpoint = isFolder ? `/api/share/folder/${itemID}` : `/api/share/file/${itemID}`
    return new Promise((resolve, reject) => {
        fetch(`/pubkey?user=${recipient}`).then(async response => {
            if (!response.ok) {
                alert("Error sharing: " + await response.text());
                reject();
                return;
            }

            crypto.ingestPublicKey((await response.json()).publicKey, async userPubKey => {
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
                        resolve(response.json());
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
    let endpoint = isFolder ? `/api/share/folder/${itemID}` : `/api/share/file/${itemID}`;

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
        window.streamSaver.mitm = "/mitm.html";
    }

    let fileStream = window.streamSaver.createWriteStream(name, {
        size: length,
    });
    return fileStream.getWriter();
}

export const uploadSendMetadata = (metadata, callback, errorCallback) => {
    uploadMetadata(metadata, sendEndpoint, callback, errorCallback);
}

export const uploadVaultMetadata = (metadata, callback, errorCallback) => {
    uploadMetadata(metadata, vaultEndpoint, callback, errorCallback);
}

export const uploadSendChunks = async (id, file, key, callback, errorCallback) => {
    await uploadChunks(sendEndpoint, id, file, key, callback, errorCallback);
}

export const uploadVaultChunks = async (id, file, key, callback, errorCallback) => {
    await uploadChunks(vaultEndpoint, id, file, key, callback, errorCallback);
}

export const downloadVaultFile = (name, download, key, callback, errorCallback) => {
    downloadFile(vaultEndpoint, name, download, key, callback, errorCallback);
}

export const downloadSentFile = (name, download, key, callback, errorCallback) => {
    downloadFile(sendEndpoint, name, download, key, callback, errorCallback);
}