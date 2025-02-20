import * as crypto from "./crypto.js";
import * as dialogs from "./dialogs/dialogs.js";
import {ActionsDialog} from "./dialogs/item_actions.js";
import {PasswordGeneratorDialog} from "./dialogs/password_generator.js";
import {VaultPassDialog} from "./dialogs/vault_pass.js";
import {ProtectedVaultDialog} from "./dialogs/protected_vault.js";
import {ShareContentDialog} from "./dialogs/share_item.js";
import * as transfer from "./transfer.js";
import * as constants from "./constants.js";
import {Endpoint, Endpoints} from "./endpoints.js";
import * as interfaces from "./interfaces.js"
import { YeetFileDB } from "./db.js";
import * as render from "./render.js";
import * as fragments from "./fragments.js";
import {VaultFolderCache} from "./cache.js";
import {ModifyVaultItem, VaultItem} from "./interfaces.js";
import {PackagedPassEntry, PassEntry} from "./strict_interfaces.js";

const gapFill = 9;
const closeFileID = "close-file";
const actionIDPrefix = "action";
const copyUsernamePrefix = "copy-username";
const copyPasswordPrefix = "copy-password";
const folderIDPrefix = "load-folder";
const itemIDPrefix = "load-item";
const sharedWithSuffix = "sharedwith";
const folderRowSuffix = "folder-row";
const fileRowSuffix = "file-row";

const emptyRow = `<tr class="blank-row"><td colspan="4"></td></tr>`
const vaultHome = `<a id="${folderIDPrefix}-" href="#">Home</a>`;
const closeFile = `<a id="${closeFileID}" href="#">Close</a>`;
const folderPlaceholder = `<a href="/vault">← Back</a> / ...`;

export enum VaultViewType {
    FileVault = 0,
    PassVault
}

enum View {
    Item = 0,
    Folder
}

type CurrentFiles = {
    [key: string]: VaultViewItem;
}

type CurrentFolders = {
    [key: string]: VaultViewFolder;
}

export class VaultViewItem {
    decName: string;
    decData: PassEntry;
    key: CryptoKey;
    [key: string]: any;

    constructor(item: interfaces.VaultItem, key: CryptoKey, decName: string, decData?: PassEntry) {
        Object.assign(this, item);
        this.key = key;
        this.decName = decName;

        if (this.passwordData.length > 0 && !decData) {
            this.decData = new PassEntry();
            this.decData.unpack(this.key, this.passwordData).then();
        } else {
            this.decData = decData;
        }
    }
}

export class VaultViewFolder {
    decName: string;
    key: CryptoKey;
    [key: string]: any;

    constructor(item: interfaces.VaultFolder, key: CryptoKey, decName: string) {
        Object.assign(this, item);
        this.key = key;
        this.decName = decName;
    }
}

export class VaultView {
    viewType: VaultViewType;
    cache: VaultFolderCache = new VaultFolderCache();

    folderDialog: HTMLDialogElement;
    passwordDialog: VaultPassDialog;
    actionsDialog: ActionsDialog;
    shareDialog: ShareContentDialog;

    folderStatus: string;
    folderID: string;
    subfolderParentID: string;

    currentItems: CurrentFiles = {};
    currentFolders: CurrentFolders = {};

    paused: boolean = false;

    folderKey: CryptoKey;
    privateKey: CryptoKey;
    publicKey: CryptoKey;

    folderEndpoint: Endpoint;
    webEndpoint: Endpoint;

    constructor(viewType: VaultViewType, privateKey: CryptoKey, publicKey: CryptoKey) {
        this.folderID = this.getFolderID();
        this.privateKey = privateKey;
        this.publicKey = publicKey;

        this.viewType = viewType;
        if (viewType === VaultViewType.FileVault) {
            this.folderEndpoint = Endpoints.VaultFolder;
            this.webEndpoint = Endpoints.HTMLVaultFolder;
        } else if (viewType === VaultViewType.PassVault) {
            this.folderEndpoint = Endpoints.PassFolder;
            this.webEndpoint = Endpoints.HTMLPassFolder;
        }
    }

    /**
     * Initializes the vault table view
     */
    initialize = () => {
        if (this.folderID.length > 0) {
            let vaultStatus = document.getElementById("vault-status");
            vaultStatus.innerHTML = folderPlaceholder;
        }

        this.loadFolder(this.folderID);

        if (this.viewType === VaultViewType.FileVault) {
            this.setupFileVaultUI();
        } else {
            this.setupPassVaultUI();
        }

        this.setupVaultDialogs();

        let newFolderBtn = document.getElementById("new-vault-folder") as HTMLButtonElement;
        this.folderDialog = document.getElementById("folder-dialog") as HTMLDialogElement;
        newFolderBtn.addEventListener("click", () => {
            let name = document.getElementById("folder-name") as HTMLInputElement;
            name.value = "";
            this.folderDialog.showModal();
        });
        this.setupFolderDialog();

        this.showStorageBar("", 0);

        document.addEventListener("click", this.clickListener, { passive: true });
    }

    /**
     * Sets up elements specific to the File Vault UI
     */
    setupFileVaultUI = () => {
        let vaultUploadBtn = document.getElementById("vault-upload") as HTMLButtonElement;
        let vaultFileInput = document.getElementById("file-input") as HTMLInputElement;
        vaultUploadBtn.addEventListener("click", () => {
            vaultFileInput.click();
        });

        vaultFileInput.addEventListener("change", this.uploadUserSelectedFiles);
        vaultFileInput.addEventListener("click touchstart", () => {
            vaultFileInput.value = "";
        });
    }

    /**
     * Sets up elements specific to the Pass Vault UI
     */
    setupPassVaultUI = () => {
        this.passwordDialog = new VaultPassDialog();
        let newPasswordBtn = document.getElementById("add-entry") as HTMLButtonElement;
        newPasswordBtn.addEventListener("click", () => {
            this.passwordDialog.create(
                this.folderID,
                this.folderKey || this.publicKey,
                this.folderKey ? crypto.encryptChunk : crypto.encryptRSA,
                this.uploadPassword);
        });

        let generatePasswordBtn = document.getElementById("generate-new-password") as HTMLButtonElement;
        generatePasswordBtn.addEventListener("click", () => {
            let pwGenDialog = new PasswordGeneratorDialog();
            pwGenDialog.show();
        });
    }

    /**
     * Sets up dialogs for the vault view
     */
    setupVaultDialogs = () => {
        this.shareDialog = new ShareContentDialog();
        this.actionsDialog = new ActionsDialog(this.#actionsCallback);
    }

    /**
     * Handles actions the user performs on individual files/folders/etc
     * @param item {VaultViewItem|VaultViewFolder} - the target folder or file
     * @param signal {Symbol} - the dialogs.DialogSignal from the actions dialog
     * @private
     */
    #actionsCallback = async (
        item: VaultViewItem|VaultViewFolder,
        signal: dialogs.DialogSignal,
    ) => {
        let id = item.refID;
        let isFolder = item instanceof VaultViewFolder;

        if (!signal) {
            return;
        }

        switch (signal) {
            case dialogs.DialogSignal.Rename:
                this.showRenameDialog(id, isFolder);
                break;
            case dialogs.DialogSignal.Download:
                this.downloadFile(id);
                break;
            case dialogs.DialogSignal.Delete:
                let confirmMsg;
                if (isFolder) {
                    confirmMsg = "Are you sure you want to delete this folder? " +
                        "This will delete all files in the folder permanently.";
                } else {
                    confirmMsg = "Are you sure you want to delete this file?";
                }

                if (confirm(confirmMsg)) {
                    dialogs.closeDialogs();
                    this.deleteVaultContent(id, item.decName, isFolder, item.refID, response => {
                        this.removeRow(id, isFolder);
                        this.removeFromCache(id, isFolder);
                    });
                }
                break;
            case dialogs.DialogSignal.Share:
                let itemKey = isFolder ?
                    this.currentFolders[id].key :
                    this.currentItems[id].key;
                let itemKeyRaw = await crypto.exportKey(itemKey, "raw");
                this.shareDialog.show(id, itemKeyRaw, isFolder, signal => {
                    if (signal === dialogs.DialogSignal.Cancel) {
                        return;
                    }

                    transfer.getSharedUsers(id, isFolder).then(response => {
                        let icon = this.generateSharedWithIcon(id,
                            response ?
                                (response as Array<JSON>).length :
                                0);
                        document.getElementById(`${id}-${sharedWithSuffix}`).innerHTML = icon;
                    });

                });
                break;
            case dialogs.DialogSignal.Remove:
                if (confirm("Are you sure you want to remove this item? " +
                    "The owner will need to re-share this with you if you need access again.")) {
                    dialogs.closeDialogs();
                    this.deleteVaultContent(id, item.decName, isFolder, item.id, () => {
                        this.removeRow(id, isFolder);
                        this.removeFromCache(id, isFolder);
                    });
                }
                break;
        }
    }

    /**
     * Uploads an encrypted password entry to the user's password vault
     * @param packaged
     */
    uploadPassword = (packaged: PackagedPassEntry) => {
        transfer.uploadVaultMetadata(packaged.upload, id => {
            let item = new interfaces.VaultItem();
            item.id = id;
            item.refID = id;
            item.name = packaged.upload.name;
            item.size = 1;
            item.modified = new Date();
            item.protectedKey = new Uint8Array(packaged.upload.protectedKey);
            item.sharedBy = "";
            item.sharedWith = 0;
            item.canModify = this.cache.get(this.folderID).folder.canModify;
            item.isOwner = this.cache.get(this.folderID).folder.isOwner;
            item.passwordData = new Uint8Array(packaged.upload.passwordData);

            let viewItem = new VaultViewItem(item, packaged.key, packaged.name, packaged.entry);

            let row = this.generateFileRow(viewItem);
            this.currentItems[id] = viewItem;
            this.cache.addItem(this.folderID, item);
            this.insertFileRow(row);
        }, () => {
            alert("Error uploading item");
        });
    }

    /**
     * Sets up event listeners for the new folder dialog.
     */
    setupFolderDialog = () => {
        let cancelUploadBtn = document.getElementById("cancel-folder");
        cancelUploadBtn.addEventListener("click", () => {
            dialogs.closeDialog(this.folderDialog);
        });

        let submitUploadBtn = document.getElementById("submit-folder");
        submitUploadBtn.addEventListener("click", () => {
            let nameInput = document.getElementById("folder-name") as HTMLInputElement;
            let folderName = nameInput.value;
            if (folderName.length === 0) {
                alert("Folder name must be > 0 characters");
                return;
            }
            this.createNewFolder(folderName).then(() => {
                dialogs.closeDialog(this.folderDialog);
            });
        });
    }

    /**
     * Grabs the folder ID segment from the current path
     * @returns {string} - The current folder ID string
     */
    getFolderID = (): string => {
        let fileVaultRoot = Endpoints.HTMLVault.path.replace("/", "")
        let passVaultRoot = Endpoints.HTMLPass.path.replace("/", "");
        let splitPath = window.location.pathname.split("/");
        for (let i = splitPath.length - 1; i > 0; i--) {
            if (splitPath[i].length > 0 &&
                splitPath[i] !== fileVaultRoot &&
                splitPath[i] !== passVaultRoot
            ) {
                return splitPath[i];
            }
        }


        return "";
    }

    /**
     * Given a vault folder ID, loads the folder contents into the table view
     * and updates the current folder ID.
     * @param newFolderID
     */
    loadFolder = async (newFolderID: string) => {
        if (this.paused) {
            return;
        }

        this.setTableLoading(true);
        this.setView(View.Folder);

        let tableBody = document.getElementById("table-body");
        tableBody.innerHTML = "";

        this.folderID = newFolderID;
        let folderPath = Endpoints.format(this.webEndpoint, this.folderID);
        window.history.pushState(this.folderID, "", folderPath);

        const loadFolderData = async (data: interfaces.VaultFolderResponse) => {
            this.emptyCurrentItems();
            this.folderKey = null;
            this.subfolderParentID = data.folder.refID;
            this.allowUploads(data.folder.canModify);
            if (!data.keySequence || data.keySequence.length === 0) {
                // In root level vault (everything is decrypted with the user's
                // private key, since content shared with them is encrypted with
                // their public key and ends up in their root folder).
                await this.loadVault(data);
            } else {
                // In sub folder, need to iterate through key sequence
                this.folderKey = await crypto.unwindKeys(this.privateKey, data.keySequence);
                await this.loadVault(data);
            }
        }

        if (this.cache.get(this.folderID)) {
            await loadFolderData(this.cache.get(this.folderID));
        } else {
            this.fetchVault(this.folderID, async (data: interfaces.VaultFolderResponse) => {
                this.cache.set(this.folderID, data);
                await loadFolderData(data);
            });
        }
    }

    /**
     * Fetches the current vault folder contents by vault folder ID. Fires a
     * callback containing the vault folder response when finished.
     * @param folderID {string}
     * @param callback {interfaces.VaultFolderResponse}
     */
    fetchVault = (
        folderID: string,
        callback: (response: interfaces.VaultFolderResponse) => void,
    ) => {
        let endpoint = Endpoints.format(this.folderEndpoint, folderID);
        fetch(endpoint)
            .then((response) => {
                return response.json();
            })
            .then((data) => {
                callback(new interfaces.VaultFolderResponse(data));
            })
            .catch((error) => {
                console.error("Error fetching vault: ", error);
            });
    }

    /**
     * Handles events when a user clicks on an element in the vault view (file,
     * folder, actions button, etc).
     * @param event {MouseEvent}
     */
    clickListener = (event: MouseEvent) => {
        event.stopPropagation();

        if (dialogs.isDialogOpen() || (event.target as Element).closest("code")) {
            return;
        }

        let id;
        let target = (event.target as HTMLElement);
        if (target.id.indexOf("-") > 0) {
            let itemIDParts = target.id.split("-");
            id = itemIDParts[itemIDParts.length - 1];
        }

        if (target.id.startsWith(actionIDPrefix)) {
            let item = this.currentFolders[id] ?
                this.currentFolders[id] :
                this.currentItems[id];
            this.actionsDialog.show(item);
        } else if (target.id.startsWith(folderIDPrefix)) {
            this.loadFolder(id);
        } else if (target.id.startsWith(itemIDPrefix)) {
            this.loadFile(id);
        } else if (target.id.startsWith(copyUsernamePrefix) ||
            target.id.startsWith(copyPasswordPrefix)) {
            if (!this.currentItems[id] && this.currentItems[id].decData) {
                return;
            }

            let passData = this.currentItems[id].decData;
            let targetValue, targetType;

            if (target.id.startsWith(copyUsernamePrefix)) {
                targetValue = passData.username;
                targetType = "Username";
            } else {
                targetValue = passData.password;
                targetType = "Password";
            }

            copyToClipboard(targetValue, success => {
                if (success) {
                    this.showStorageBar(`${targetType} copied!`, 0);
                }
            });
        } else if (target.id === closeFileID) {
            this.closeFileView();
        }
    }

    /**
     * Initiates file upload after files are selected by the user
     */
    uploadUserSelectedFiles = async () => {
        let vaultFileInput = document.getElementById("file-input") as HTMLInputElement;
        let currentFile = 0;
        let totalFiles = vaultFileInput.files.length;

        const startUpload = async idx => {
            await this.uploadFile(
                vaultFileInput.files[idx],
                idx,
                totalFiles,
                async (success, file, view ) => {
                    if (success) {
                        let row = this.generateFileRow(view);
                        this.currentItems[file.id] = view;
                        this.cache.addItem(this.folderID, file);
                        this.insertFileRow(row);
                        if (idx < totalFiles - 1) {
                            await startUpload(idx + 1);
                        }
                    }
                });
        }

        await startUpload(currentFile);
    }

    /**
     * Update the UI to allow or disallow uploading to the current folder.
     * @param allow {boolean} - True to allow, false to disallow
     */
    allowUploads = (allow: boolean): void => {
        let uploadBtn = document.getElementById("vault-upload") as HTMLButtonElement;
        let addBtn = document.getElementById("add-entry") as HTMLButtonElement;
        let folderBtn = document.getElementById("new-vault-folder") as HTMLButtonElement;

        if (uploadBtn) {
            uploadBtn.disabled = !allow;
        }

        if (addBtn) {
            addBtn.disabled = !allow;
        }

        folderBtn.disabled = !allow;
    }

    /**
     * Uploads one or multiple files, indicating progress to the user.
     * @param file {File} - The file to upload
     * @param idx {number} - The index of the file being uploaded (if multiple)
     * @param total {number} - The total number of files being uploaded
     * @param callback {function(boolean, VaultItem, VaultViewItem)} - A callback
     * indicating if the upload was successful, and if so, the file and view class
     * for that file
     */
    uploadFile = async (
        file: File,
        idx: number,
        total: number,
        callback: (success: boolean, item: interfaces.VaultItem, file: VaultViewItem) => void,
    ) => {
        this.showFileIndicator("");

        if (total > 1) {
            this.setVaultMessage(`Uploading ${file.name}... (${idx + 1} / ${total})`);
        } else {
            this.setVaultMessage(`Uploading ${file.name}...`);
        }

        this.paused = true;

        let key = await crypto.generateRandomKey();
        let protectedKey = await this.encryptData(key);
        let importedKey = await crypto.importKey(key);

        let encryptedName = await crypto.encryptString(importedKey, file.name);
        let hexName = toHexString(encryptedName);
        let metadata = new interfaces.VaultUpload({
            name: hexName,
            length: file.size,
            chunks: getNumChunks(file.size),
            folderID: this.folderID,
            protectedKey: Array.from(protectedKey),
        });

        transfer.uploadVaultMetadata(metadata, id => {
            transfer.uploadVaultChunks(id, file, importedKey, finished => {
                this.paused = !finished;
                if (finished) {
                    let size = this.cache.get(this.folderID).folder.isOwner ? file.size : 0;
                    if (idx + 1 === total) {
                        let uploadName = "files";
                        if (total === 1) {
                            uploadName = file.name;
                        }

                        this.showStorageBar(`Finished uploading ${uploadName}!`, size);
                    } else {
                        this.showStorageBar("", size);
                    }

                    let item = new interfaces.VaultItem();
                    item.id = id;
                    item.refID = id;
                    item.name = hexName;
                    item.size = file.size + constants.TotalOverhead;
                    item.modified = new Date();
                    item.protectedKey = protectedKey;
                    item.sharedBy = "";
                    item.sharedWith = 0;
                    item.canModify = this.cache.get(this.folderID).folder.canModify;
                    item.isOwner = this.cache.get(this.folderID).folder.isOwner;

                    let view = new VaultViewItem(item, importedKey, file.name);
                    callback(true, item, view);
                }
            }, errorMessage => {
                this.paused = false;
                callback(false, undefined, undefined);
                alert(errorMessage);
                this.showStorageBar("", 0);
            });
        }, () => {
            this.paused = false;
            callback(false, undefined, undefined);
            this.showStorageBar("", 0);
        });
    }

    /**
     * Decrypt encrypted file/folder data using either RSA (root folder) or AES
     * (any subfolder)
     * @param data {Uint8Array} - The data to decrypt
     * @returns {Promise<Uint8Array>} - The decrypted chunk of data
     */
    decryptData = async (data: Uint8Array): Promise<Uint8Array> => {
        if (!this.folderKey) {
            return await crypto.decryptRSA(this.privateKey, data);
        } else {
            return await crypto.decryptChunk(this.folderKey, data);
        }
    }

    /**
     * Encrypt file/folder data using either RSA (root folder only) or AES (any
     * subfolder)
     * @param data {Uint8Array} - The data to encrypt
     */
    encryptData = async (data: Uint8Array): Promise<Uint8Array> => {
        if (!this.folderKey) {
            return await crypto.encryptRSA(this.publicKey, data);
        } else {
            return await crypto.encryptChunk(this.folderKey, data);
        }
    }

    /**
     * Download a vault file by file ID
     * @param id {string} - The ID of the file to download
     */
    downloadFile = (id: string): void => {
        if (this.paused) {
            return;
        }

        this.paused = true;
        this.showFileIndicator("");
        this.setVaultMessage("Downloading...");

        let xhr = new XMLHttpRequest();
        let url = Endpoints.format(Endpoints.DownloadVaultFileMetadata, id);
        xhr.open("GET", url, true);
        xhr.setRequestHeader("Content-Type", "application/json");

        xhr.onreadystatechange = async () => {
            if (xhr.readyState === 4 && xhr.status === 200) {
                let download = new interfaces.VaultDownloadResponse(xhr.responseText);
                let itemKey = await this.decryptData(download.protectedKey);
                let tmpKey = await crypto.importKey(itemKey);
                let name = await crypto.decryptString(tmpKey, hexToBytes(download.name));

                this.setVaultMessage(`Downloading ${name}...`);

                transfer.downloadVaultFile(name, download, tmpKey, finished => {
                    this.paused = !finished;
                    if (finished) {
                        this.showStorageBar("", 0);
                    }
                }, () => {
                    alert("Error downloading file!");
                });
            } else if (xhr.readyState === 4 && xhr.status !== 200) {
                alert(`Error ${xhr.status}: ${xhr.responseText}`);
                this.showStorageBar("", 0);
            }
        };

        xhr.send();
    }

    setTableLoading = (loading: boolean) => {
        let loadingHeader = document.getElementById("loading-header");
        let tableHeader = document.getElementById("table-header");

        if (loading) {
            loadingHeader.className = "visible";
            tableHeader.className = "hidden";
        } else {
            loadingHeader.className = "hidden";
            tableHeader.className = "visible";
        }
    }

    setView = (view: View) => {
        let fileDiv = document.getElementById("vault-file-div");
        let folderDiv = document.getElementById("vault-items-div");

        if (view === View.Item) {
            fileDiv.className = "visible";
            folderDiv.className = "hidden";
        } else if (view === View.Folder) {
            fileDiv.innerHTML = fragments.VaultFileViewDiv();
            fileDiv.className = "hidden";
            folderDiv.className = "visible";
        }
    }

    /**
     * Update the status bar with new HTML
     * @param contents {string}
     */
    setStatus = (contents: string) => {
        let vaultStatus = document.getElementById("vault-status");
        vaultStatus.innerHTML = contents;
    }

    /**
     * Resets the decrypted file and folder cache
     */
    emptyCurrentItems = () => {
        this.currentItems = {};
        this.currentFolders = {};
    }

    /**
     * Displays file metadata and displays a preview of the file.
     * @param fileID
     */
    loadFile = (fileID: string) => {
        if (this.paused) {
            return;
        }

        let file = this.currentItems[fileID];
        if (!file) {
            alert("Unable to open file!");
            this.loadFolder(this.folderID);
            return;
        }

        if (file.decData) {
            this.passwordDialog.edit(
                file.decName,
                file.decData,
                file.key,
                (packaged) =>
                {
                    this.modifyItem(fileID, false, packaged.name, packaged.encData).then();
                }, file.canModify);
            return;
        }

        this.setStatus(closeFile);
        this.setView(View.Item);

        let headerDiv = document.getElementById("vault-file-header");
        let htmlDiv = document.getElementById("vault-file-content");
        let textOnly = document.getElementById("vault-text-content");
        let textWrapper = document.getElementById("vault-text-wrapper");

        headerDiv.innerHTML = "";
        htmlDiv.innerHTML = "";
        textOnly.innerText = "";

        this.generateItemView(file, header => {
            headerDiv.innerHTML = header;
        }, html => {
            htmlDiv.innerHTML = html;
            textWrapper.className = "hidden";
            htmlDiv.className = "visible";
        }, text => {
            textOnly.innerText = text;
            htmlDiv.className = "hidden";
            textWrapper.className = "visible";
        });
    }

    /**
     * Closes the file viewer, re-opens the folder view, and resets the status
     * bar to the current folder status.
     */
    closeFileView = () => {
        let fileText = document.getElementById("vault-text-content");
        fileText.innerText = "";
        this.setView(View.Folder);
        this.setStatus(this.folderStatus);
    }

    /**
     * Uses a VaultFolderResponse to load the contents into the view.
     * @param data {interfaces.VaultFolderResponse}
     */
    loadVault = async (data: interfaces.VaultFolderResponse) => {
        this.emptyCurrentItems();

        if (data.folder.name.length > 0) {
            crypto.decryptString(this.folderKey, hexToBytes(data.folder.name)).then(folderName => {
                let folderLink = `${vaultHome} | <a id="${folderIDPrefix}-${data.folder.parentID}" href="#">← Back</a> / ${folderName}`;
                this.folderStatus = folderLink;
                this.setStatus(folderLink);
            }).catch(() => {
                this.setStatus("[decryption error]");
            });
        } else {
            this.setStatus(vaultHome);
            this.folderStatus = vaultHome;
        }

        let tableBody = document.getElementById("table-body");
        let folders = data.folders;
        let items = data.items;

        for (let i = 0; i < folders.length; i++) {
            let folder = folders[i];
            let subFolderKey = await this.decryptData(folder.protectedKey);
            let tmpKey = await crypto.importKey(subFolderKey);
            let decName = await crypto.decryptString(tmpKey, hexToBytes(folder.name));

            let vaultFolder = new VaultViewFolder(folder, tmpKey, decName);
            this.currentFolders[folder.refID] = vaultFolder;
            let row = this.generateFolderRow(vaultFolder);
            tableBody.innerHTML += row;
        }

        for (let i = 0; i < items.length; i++) {
            let item = items[i];
            let itemKey = await this.decryptData(item.protectedKey);
            let tmpKey = await crypto.importKey(itemKey);
            let decName = await crypto.decryptString(tmpKey, hexToBytes(item.name));

            let decEntry;
            if (item.passwordData && item.passwordData.length > 0) {
                let decData = await crypto.decryptChunk(tmpKey, item.passwordData);
                decEntry = new PassEntry(new TextDecoder().decode(decData));
            }

            let vaultFile = new VaultViewItem(item, tmpKey, decName, decEntry);
            this.currentItems[item.refID] = vaultFile;
            let row = this.generateFileRow(vaultFile);
            tableBody.innerHTML += row;
        }

        for (let i = 0; i < gapFill - (folders.length + items.length); i++) {
            tableBody.innerHTML += emptyRow;
        }

        this.setTableLoading(false);
    }

    /**
     * Prepends an HTML string `tr` element to the vault table body. Note that this
     * should only be used for folders, since new ones should always be at the top.
     * @param row {string}
     */
    addTableRow = (row: string) => {
        let tableBody = document.getElementById("table-body");
        tableBody.innerHTML = row + tableBody.innerHTML;
    }

    /**
     * Inserts an HTML string `tr` element into the vault table element below all
     * folders but above any existing files.
     * @param row {string}
     */
    insertFileRow = (row: string) => {
        let tableBody = document.getElementById("table-body");
        let rows = tableBody.getElementsByTagName("tr");

        let tempContainer = document.createElement("tbody");
        tempContainer.innerHTML = row;
        let rowNode = tempContainer.firstElementChild;

        if (rows.length === 0) {
            tableBody.innerHTML = row;
            return;
        }

        for (let i = 0; i < rows.length; i++) {
            if (!rows[i].id.endsWith(fileRowSuffix) && rows[i].className !== "blank-row") {
                continue;
            }

            tableBody.insertBefore(rowNode, rows[i]);
            return;
        }

        tableBody.innerHTML += row;
    }

    /**
     * Creates an HTML `tr` element string using properties of the provided
     * VaultViewFolder element.
     * @param folder {VaultViewFolder}
     */
    generateFolderRow = (folder: VaultViewFolder) => {
        let classes = folder.sharedBy.length > 0 ? "shared-link" : "folder-link";
        let link = `<a id="${folderIDPrefix}-${folder.refID}" class="${classes}" href="#">${folder.decName}/</a>`
        return this.generateRow(link, folder);
    }

    /**
     * Creates an HTML `tr` element string using properties of the provided
     * VaultViewItem element.
     * @param file {VaultViewItem}
     */
    generateFileRow = (file: VaultViewItem) => {
        let classes = file.sharedBy.length > 0 ? "shared-link" : "file-link";
        let id = `${itemIDPrefix}-${file.refID}`;
        let link = `<a data-testid="${id}" id="${id}" class="${classes}" href="#">${file.decName}</a>`
        return this.generateRow(link, file);
    }

    /**
     * Generates an HTML string `tr` element for a folder or item.
     * @param link {string} - The `a` tag for the vault item
     * @param item {VaultViewItem|VaultViewFolder} - The item or folder to use
     * for generating the table row.
     */
    generateRow = (link: string, item: VaultViewItem|VaultViewFolder) => {
        let id = item.refID;
        let modified = formatDate(item.modified);
        let sizeOrCopy = "";
        let suffix;

        let iconSrc;
        let iconClasses = "small-icon";
        if (item.sharedBy) {
            iconClasses += " shared-icon";
        }

        if (item instanceof VaultViewFolder) {
            suffix = folderRowSuffix;
            iconClasses += " accent-icon";
            iconSrc = "/static/icons/folder.svg";
        } else if (item.passwordData && item.passwordData.length > 0) {
            suffix = fileRowSuffix;
            iconSrc = "/static/icons/key.svg";
            sizeOrCopy = `
            <img id="${copyUsernamePrefix}-${id}" title="Copy Username" alt="Copy Username" class="small-icon accent-icon" src="/static/icons/user.svg">
            <img id="${copyPasswordPrefix}-${id}" title="Copy Password" alt="Copy Password" class="small-icon accent-icon" src="/static/icons/asterisk.svg">`;
        } else {
            suffix = fileRowSuffix;
            sizeOrCopy = calcFileSize(item.size - constants.TotalOverhead);
            iconSrc = "/static/icons/file.svg";
        }

        let itemIcon = `<img class="${iconClasses}" src="${iconSrc}">`;

        let sharedIcon = this.generateSharedWithIcon(id, item.sharedWith);
        let sharedByIcon = "";
        if (item.sharedBy) {
            sharedByIcon = `
            <br>
            <img class="small-icon shared-icon" src="/static/icons/shared.svg">
            &nbsp;${item.sharedBy}`;
        }

        if (item.decData && (item.decData.username || item.decData.urls.length > 0)) {
            if (item.decData.username) {
                sharedByIcon += `<br>
                <span class="secondary-text"><img class="small-icon secondary-icon" src="/static/icons/user.svg">
                ${item.decData.username}</span>`;
            }

            if (item.decData.urls.length > 0 && item.decData.urls[0].length > 0) {
                let url = item.decData.urls[0];
                if (item.decData.urls.length > 1) {
                    url += ` (+${item.decData.urls.length - 1})`;
                }
                sharedByIcon += `<br>
                <span class="secondary-text"><img class="small-icon secondary-icon" src="/static/icons/web.svg">
                ${url}</span>`;
            }
        }

        let idStr = `${actionIDPrefix}-${id}`;

        return `
        <tr id="${id}-${suffix}">
            <td><span>${itemIcon} ${link} ${sharedIcon}</span> ${sharedByIcon}</td>
            <td>${sizeOrCopy}</td>
            <td id="${id}-modified">${modified}</td>
            <td class="action-icon" data-testid=${idStr} id="${idStr}">⋮</td>
        </tr>`
    }

    /**
     * Generates an icon indicating the number of users a file/folder has been
     * shared with.
     * @param id {string}
     * @param sharedWithCount {number}
     */
    generateSharedWithIcon = (id: string, sharedWithCount: number) => {
        let div = `<div class="shared-with-div" id="${id}-${sharedWithSuffix}">`
        if (sharedWithCount === 0) {
            return div + "</div>";
        }

        return `${div}<img class="small-icon" src="/static/icons/share.svg"> (${sharedWithCount})</div>`
    }

    /**
     * Generates a view of the selected VaultViewItem, displaying the contents in
     * a temporary div. Provides three callbacks for the caller:
     * - header: The header content to display above the file
     * - html: The HTML representation of the file (if applicable)
     * - text: The plaintext representation of the file (if applicable)
     *
     * The `html` callback is used if the file is an image, audio file, pdf, etc.
     * Otherwise, the `text` callback is used.
     * @param file {VaultViewItem}
     * @param header {callback(string)}
     * @param html {callback(string)}
     * @param text {callback(string)}
     */
    generateItemView = (
        file: VaultViewItem,
        header: (string) => void,
        html: (string) => void,
        text: (string) => void,
    ) => {
        let name = file.decName;
        let date = formatDate(file.modified);
        let size = calcFileSize(file.size - constants.TotalOverhead);
        let htmlHead = `<p class="no-top-margin">${name} | ${size} | ${date}</p><hr>`;
        header(htmlHead+fragments.LoadingSpinner("sub"));

        let url = Endpoints.format(Endpoints.DownloadVaultFileMetadata, file.refID);
        fetch(url).then(response => {
            header(htmlHead);
            if (!response.ok) {
                text("Failed to fetch file");
                return;
            }

            response.json().then(json => {
                let metadata = new interfaces.VaultDownloadResponse(json);
                if (metadata.chunks > 5) {
                    text("File too large to preview");
                    return;
                }

                let bytes;
                const fetchChunk = (chunkNum: number) => {
                    let endpoint = Endpoints.DownloadVaultFileData;
                    let chunkURL = Endpoints.format(endpoint, metadata.id, `${chunkNum}`);
                    transfer.fetchSingleChunk(chunkURL, file.key, chunk => {
                        if (!bytes) {
                            bytes = chunk;
                        } else {
                            let combinedBuffer = new ArrayBuffer(chunk.byteLength + bytes.byteLength);
                            let combinedArray = new Uint8Array(combinedBuffer);

                            combinedArray.set(new Uint8Array(bytes), 0);
                            combinedArray.set(new Uint8Array(chunk), bytes.byteLength);
                            bytes = combinedArray;
                        }

                        if (chunkNum === metadata.chunks) {
                            if (render.isNonTextFileType(file.decName)) {
                                render.renderFileHTML(file.decName, bytes, (tag, url) => {
                                    html(tag);
                                    if (url) {
                                        let download = `<a href="${url}" download="${name}">Download</a>`;
                                        let popOut = `<a target="_blank" href="${url}">Pop Out</a>`;
                                        this.setStatus(`${closeFile} | ${popOut} | ${download}`);
                                    } else {
                                        this.setStatus(`${closeFile}`);
                                    }
                                });
                            } else {
                                try {
                                    let textContent = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
                                    text(textContent);
                                } catch (_) {
                                    text("Cannot show file preview");
                                }
                            }
                        } else {
                            fetchChunk(chunkNum + 1);
                        }
                    }, () => {
                        text("Error downloading file content");
                    });
                }

                fetchChunk(1);
            })
        });
    }

    /**
     * Updates the name of a file/folder
     * @param id {string} - The ID of the modified content
     * @param isFolder {boolean} - True if a folder, else false
     * @param name {string} - The new (unencrypted) name
     */
    updateRow = (id: string, isFolder: boolean, name: string) => {
        let prefix = isFolder ? folderIDPrefix : itemIDPrefix;
        let nameID = `${prefix}-${id}`;
        let modifiedID = `${id}-modified`;
        document.getElementById(nameID).innerText = name + (isFolder ? "/" : "");
        document.getElementById(modifiedID).innerText = formatDate(Date());
    }

    /**
     * Removes a file or folder row from the vault view table.
     * @param id {string} - The file/folder ID
     * @param isFolder {boolean} - True if a folder, else false
     */
    removeRow = (id: string, isFolder: boolean) => {
        let rowID = `${id}-${isFolder ? folderRowSuffix : fileRowSuffix}`;
        document.getElementById(rowID).remove();
    }

    /**
     * Removes a file or folder from the local folder cache.
     * @param id {string}
     * @param isFolder {boolean}
     */
    removeFromCache = (id: string, isFolder: boolean) => {
        if (isFolder) {
            this.cache.updateFolders(this.folderID, this.cache.get(this.folderID).folders.filter(obj => {
                return obj.id !== id;
            }));
        } else {
            this.cache.updateItems(this.folderID, this.cache.get(this.folderID).items.filter(obj => {
                return obj.id !== id;
            }));
        }
    }

    /**
     * Displays the actions dialog for a file or folder
     * @param id {string}
     */
    showActionsDialog = (id: string) => {
        let isFolder = false;
        let actionsDialog = document.getElementById("actions-dialog") as HTMLDialogElement;
        let title = document.getElementById("actions-dialog-title") as HTMLHeadingElement;
        let item;

        if (this.currentFolders[id]) {
            item = this.currentFolders[id];
            title.innerText = "Folder: " + item.decName;
            isFolder = true;
        } else if (this.currentItems[id]) {
            item = this.currentItems[id];
            title.innerText = item.decName;
        }

        let actionDownload = document.getElementById("action-download");
        actionDownload.style.display = isFolder ? "none" : "flex";
        actionDownload.addEventListener("click", event => {
            event.stopPropagation();
            dialogs.closeDialog(actionsDialog);
            this.downloadFile(id);
        });

        // TOmaybeDO
        let actionSend = document.getElementById("action-send");
        //actionSend.style.display = isFolder ? "none" : "flex";
        actionSend.style.display = "none";
        actionSend.addEventListener("click", event => { });

        let actionRename = document.getElementById("action-rename");
        if (item.canModify) {
            actionRename.style.display = "flex";
            actionRename.addEventListener("click", (event) => {
                event.stopPropagation();
                this.showRenameDialog(id, isFolder);
                dialogs.closeDialog(actionsDialog);
            });
        } else {
            actionRename.style.display = "none";
        }

        let actionLink = document.getElementById("action-link");
        actionLink.style.display = "none";
        // if (item.isOwner) {
        //     actionLink.style.display = "flex";
        //     actionLink.addEventListener("click", event => {
        //         event.stopPropagation();
        //         showLinkDialog(id, isFolder);
        //         dialogs.closeDialog(actionsDialog);
        //     });
        // }

        let actionShare = document.getElementById("action-share");
        let shareDialog = new ShareContentDialog();
        if (item.isOwner) {
            actionShare.style.display = "flex";
            actionShare.addEventListener("click", async event => {
                event.stopPropagation();

                dialogs.closeDialog(actionsDialog);
            });
        } else {
            actionShare.style.display = "none";
        }

        let actionDelete = document.getElementById("action-delete");
        if (item.isOwner) {
            actionDelete.style.display = "flex";
            actionDelete.addEventListener("click", event => {
                event.stopPropagation();


            });
        } else {
            actionDelete.style.display = "none";
        }

        let actionRemove = document.getElementById("action-remove");
        if (!item.isOwner && this.cache.get(this.folderID).folder.canModify) {
            actionRemove.style.display = "flex";
            actionRemove.addEventListener("click", event => {
                event.stopPropagation();

            });
        } else {
            actionRemove.style.display = "none";
        }

        let cancel = document.getElementById("cancel-action");
        cancel.addEventListener("click", () => dialogs.closeDialog(actionsDialog));

        actionsDialog.showModal();
    }

    /**
     * Displays the renaming dialog for a file or folder
     * @param id {string}
     * @param isFolder {boolean}
     */
    showRenameDialog = (id: string, isFolder: boolean) => {
        let renameDialog = document.getElementById("rename-dialog") as HTMLDialogElement;
        let dialogTitle = document.getElementById("rename-title") as HTMLHeadingElement;
        dialogTitle.innerText = isFolder ? "Rename Folder" : "Rename File";

        let newNameInput = document.getElementById("new-name") as HTMLInputElement;
        newNameInput.value = isFolder ?
            this.currentFolders[id].decName :
            this.currentItems[id].decName;

        let cancelBtn = document.getElementById("cancel-rename");
        cancelBtn.addEventListener("click", () => {
            dialogs.closeDialog(renameDialog);
        });

        let submitBtn = document.getElementById("submit-rename") as HTMLButtonElement;
        submitBtn.addEventListener("click", async () => {
            submitBtn.disabled = true;
            await this.modifyItem(id, isFolder, newNameInput.value);
        });

        renameDialog.showModal();
    }

    /**
     * Modifies a file or folder with a new name or password content
     * @param id {string}
     * @param isFolder {boolean}
     * @param newName {string}
     * @param newData? {Uint8Array}
     */
    modifyItem = async (id: string, isFolder: boolean, newName: string, newData?: Uint8Array) => {
        let key;
        if (isFolder) {
            key = this.currentFolders[id].key;
        } else {
            key = this.currentItems[id].key;
        }

        let newNameEncrypted = await crypto.encryptString(key, newName);
        let hexName = toHexString(newNameEncrypted);

        let modify = new ModifyVaultItem();
        modify.name = hexName;
        if (newData) {
            modify.passwordData = newData;
        }

        let endpoint = isFolder ?
            Endpoints.format(this.folderEndpoint, id) :
            Endpoints.format(Endpoints.VaultFile, id);
        fetch(endpoint, {
            method: "PUT",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(modify, jsonReplacer)
        }).then(response => {
            if (response.ok) {
                dialogs.closeDialogs();
                if (isFolder) {
                    this.currentFolders[id].name = hexName;
                    this.currentFolders[id].decName = newName;
                    this.updateRow(id, isFolder, newName);
                } else {
                    this.currentItems[id].name = hexName;
                    this.currentItems[id].decName = newName;
                    this.updateRow(id, isFolder, newName);
                }

                if (!newData) {
                    this.showActionsDialog(id);
                }
            } else {
                alert("Error renaming file");
            }
        }).catch(error => {
            alert("Error renaming file: " + error);
        });
    }

    /**
     * Deletes a file or folder from the user's vault permanently
     * @param id {string} - The file/folder ID to delete
     * @param name {string} - The unencrypted name of the content to be deleted
     * @param isFolder {boolean} - True if a folder, else false
     * @param sharedID {string} - The reference ID of the file/folder
     * @param callback {(DeleteResponse)}
     */
    deleteVaultContent = (
        id: string,
        name: string,
        isFolder: boolean,
        sharedID: string,
        callback: (resp: interfaces.DeleteResponse) => void,
    ) => {
        let modID = sharedID !== id ? sharedID : id;
        let endpoint = isFolder ?
            Endpoints.format(this.folderEndpoint, modID) :
            Endpoints.format(Endpoints.VaultFile, modID);

        this.paused = true;
        this.showFileIndicator(`Deleting ${name}...`);

        let sharedParam = sharedID !== id ? "?shared=true" : "";

        fetch(`${endpoint}${sharedParam}`, {
            method: "DELETE"
        }).then(response => {
            this.paused = false;
            if (response.ok) {
                response.json().then(json => {
                    let resp = new interfaces.DeleteResponse(json);
                    let freed = this.cache.get(this.folderID).folder.isOwner ?
                        -resp.freedSpace :
                        0;
                    this.showStorageBar(`Deleted ${name}!`, freed);
                    callback(resp);
                })
            } else {
                alert("Error deleting item");
                this.showStorageBar("", 0);
            }
        }).catch(error => {
            alert("Error deleting item: " + error);
        });
    }

    /**
     * Creates a new folder inside the current vault folder
     * @param folderName {string}
     */
    createNewFolder = async (folderName: string) => {
        let xhr = new XMLHttpRequest();
        xhr.open("POST", Endpoints.format(this.folderEndpoint, ""), false);
        xhr.setRequestHeader("Content-Type", "application/json");

        let newFolderKey = await crypto.generateRandomKey();
        let newFolderKeyImported = await crypto.importKey(newFolderKey);
        let encFolderKey = await this.encryptData(newFolderKey);
        let encName = await crypto.encryptString(newFolderKeyImported, folderName);
        let encNameEncoded = toHexString(encName);

        xhr.onreadystatechange = async () => {
            if (xhr.readyState === 4 && xhr.status === 200) {
                let response = new interfaces.NewFolderResponse(xhr.responseText);
                let vaultFolder = new interfaces.VaultFolder();
                vaultFolder.id = response.id;
                vaultFolder.refID = response.id;
                vaultFolder.name = encNameEncoded;
                vaultFolder.modified = new Date();
                vaultFolder.protectedKey = encFolderKey;
                vaultFolder.isOwner = this.cache.get(this.folderID).folder.isOwner;
                vaultFolder.canModify = this.cache.get(this.folderID).folder.canModify;
                vaultFolder.sharedBy = "";
                vaultFolder.sharedWith = 0;

                let vaultViewFolder = new VaultViewFolder(
                    vaultFolder,
                    newFolderKeyImported,
                    folderName);
                this.currentFolders[response.id] = vaultViewFolder;
                let row = this.generateFolderRow(vaultViewFolder);
                this.cache.addFolder(this.folderID, vaultFolder);
                this.addTableRow(row);
            } else if (xhr.readyState === 4 && xhr.status !== 200) {
                alert(`Error ${xhr.status}: ${xhr.responseText}`);
            }
        };

        xhr.send(JSON.stringify({
            name: encNameEncoded,
            parentID: this.folderID,
            protectedKey: Array.from(encFolderKey),
        }));
    }

    /**
     * Updates the storage indicator on the vault page with a change in storage
     * amount, or a temporary message.
     * @param tmpMessage {string} - The temporary message to display to the user
     * @param usedStorageDiff {number} - The difference in used storage (can be negative)
     * @returns {void}
     */
    showStorageBar = (tmpMessage: string, usedStorageDiff: number): void => {
        let storageBar = document.getElementById("storage-bar") as HTMLProgressElement;
        let itemBar = document.getElementById("item-bar") as HTMLProgressElement;
        let vaultMessage = document.getElementById("vault-message") as HTMLSpanElement;

        storageBar.style.display = "inherit";
        itemBar.style.display = "none";

        if (usedStorageDiff) {
            storageBar.value += usedStorageDiff;
        }

        let barTitle, barText;
        let isUnlimited = storageBar.max === 1;
        if (isUnlimited) {
            storageBar.value = storageBar.max;
            barText = "Unlimited";
        } else {
            let used, available;
            if (this.viewType === VaultViewType.PassVault) {
                barTitle = "Items";
                available = storageBar.max;
                used = storageBar.value;
            } else {
                barTitle = "Storage";
                available = calcFileSize(storageBar.max);
                used = calcFileSize(storageBar.value);
            }

            barText = `${barTitle}: ${used} / ${available}`;
        }

        if (tmpMessage) {
            vaultMessage.innerText = tmpMessage;
            setTimeout(() => vaultMessage.innerText = barText, 3000);
        } else {
            vaultMessage.innerText = barText;
        }
    }

    /** Hides the storage bar and displays a file upload indicator in its place.
     * @param msg {string} - A message to display alongside the indicator
     * @returns {void}
     */
    showFileIndicator = (msg: string) => {
        let storageBar = document.getElementById("storage-bar");
        let itemBar = document.getElementById("item-bar");

        storageBar.style.display = "none";
        itemBar.style.display = "inherit";

        if (msg) {
            let vaultMessage = document.getElementById("vault-message");
            vaultMessage.innerText = msg;
        }
    }

    /**
     * Updates the vault message below the progress bar
     * @param msg {string}
     */
    setVaultMessage = (msg: string) => {
        let vaultMessage = document.getElementById("vault-message");
        vaultMessage.innerHTML = `<img class="small-icon progress-spinner" src="/static/icons/progress.svg">${msg}`;
    }
}

/**
 * Prepares the vault for usage by grabbing the user's vault key pair
 * @param callback
 */
export const prep = (callback: (privKey: CryptoKey, pubKey: CryptoKey) => void) => {
    let vaultPassDialog = new ProtectedVaultDialog();
    let yeetfileDB = new YeetFileDB();
    yeetfileDB.isPasswordProtected(isProtected => {
        if (isProtected) {
            vaultPassDialog.show(yeetfileDB, (privKey, pubKey) => {
                callback(privKey, pubKey);
            }, null);
        } else {
            yeetfileDB.getVaultKeyPair("", false)
                .then(async ([privKey, pubKey]) => {
                    callback(privKey as CryptoKey, pubKey as CryptoKey);
                })
                .catch(e => {
                    console.error(e);
                    alert(e);
                });
        }
    });
};