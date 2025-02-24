import * as crypto from "./crypto.js";
import * as interfaces from "./interfaces.js";
import * as localstorage from "./localstorage.js";
import * as transfer from "./transfer.js";
import {Endpoints} from "./endpoints.js";

type SendForm = {
    files: FileList,
    password: string,
    passwordConfirm: string,
    downloads: number,
    expiration: number,
    expUnits: ExpUnits,
    text: string,
}

const init = () => {
    setupTypeToggles();
    setupCopyButton();
    setupInsecureLinkButton();
    fetchSendSettings();
    updateProgressBar();

    let usePasswordCB = document.getElementById("use-password") as HTMLInputElement;
    let passwordInput = document.getElementById("password") as HTMLInputElement;
    let confirmPasswordInput = document.getElementById("confirm-password") as HTMLInputElement;
    let showPasswordCB = document.getElementById("show-password") as HTMLInputElement;
    let passwordDiv = document.getElementById("password-div") as HTMLDivElement;

    showPasswordCB.addEventListener("change", (event) => {
        let target = event.currentTarget as HTMLInputElement;
        passwordInput.type = target.checked ? "text" : "password";
        confirmPasswordInput.type = target.checked ? "text" : "password";
    });

    usePasswordCB.addEventListener("change", (event) => {
        let target = event.currentTarget as HTMLInputElement;
        passwordInput.disabled = !target.checked;
        confirmPasswordInput.disabled = !target.checked;
        showPasswordCB.disabled = !target.checked;

        if (target.checked) {
            passwordDiv.style.display = "inline";
        } else if (window.innerWidth <= 425 && !target.checked) {
            passwordDiv.style.display = "none";
        }

        if (!target.checked) {
            passwordInput.value = "";
            passwordInput.type = "password";
            confirmPasswordInput.value = "";
            confirmPasswordInput.type = "password";
            showPasswordCB.checked = false;
        }
    });

    let uploadTextContent = document.getElementById("upload-text-content") as HTMLInputElement;
    let uploadTextLabel = document.getElementById("upload-text-label");
    uploadTextLabel.innerText=`Text (${uploadTextContent.value.length}/2000):`;
    uploadTextContent.addEventListener("input", () => {
        if (uploadTextLabel) {
            uploadTextLabel.innerText=`Text (${uploadTextContent.value.length}/2000):`;
        }
    });

    let form = document.getElementById("upload-form") as HTMLFormElement;
    let nameDiv = document.getElementById("name-div") as HTMLDivElement;
    let filePicker = document.getElementById("upload") as HTMLInputElement;
    filePicker.addEventListener("change", () => {
        if (filePicker.files.length > 1) {
            nameDiv.style.display = "inherit";
        } else {
            nameDiv.style.display = "none";
        }
    });

    form.addEventListener("reset", (event) => {
        resetForm();
    });

    form.addEventListener("submit", async (event) => {
        event.preventDefault();

        let formValues = getFormValues();

        if (validateForm(formValues)) {
            setFormEnabled(false);
            updateProgress("Initializing...");
            let [key, salt] = await crypto.deriveSendingKey(
                formValues.password,
                undefined);

            let rawKey = await crypto.exportKey(key, "raw");
            let keyHex = toURLSafeBase64(rawKey);
            let fileSecret = formValues.password.length > 0 ?
                toURLSafeBase64(salt) : // file has password, share file w/ salt only
                keyHex; // file has no password, share w/ hex key

            if (isFileUpload()) {
                if (formValues.files.length > 1) {
                    alert("Feature not supported");
                    // await submitFormMulti(formValues, key, salt, allowReset);
                } else {
                    await submitFormSingle(formValues, key, fileSecret, allowReset);
                }
            } else {
                await submitFormText(formValues, key, fileSecret, allowReset);
            }
        }
    });
}

const fetchSendSettings = () => {
    let downloads = document.getElementById("downloads") as HTMLInputElement;
    let expiration = document.getElementById("expiration") as HTMLInputElement;
    let expUnits = document.getElementById("duration-unit") as HTMLSelectElement;

    downloads.valueAsNumber = localstorage.getDefaultSendDownloads();
    expiration.valueAsNumber = localstorage.getDefaultSendExpiration();
    expUnits.selectedIndex = localstorage.getDefaultSendExpirationUnits();
}

const setFormEnabled = on => {
    let fieldset = document.getElementById("form-fieldset") as HTMLFieldSetElement;
    fieldset.disabled = !on;
}

const updateProgress = (txt) => {
    let uploadBtn = document.getElementById("submit") as HTMLButtonElement;
    uploadBtn.disabled = true;
    uploadBtn.value = txt;
}

const allowReset = () => {
    updateProgress("Done!")
    let reset = document.getElementById("reset");
    reset.style.display = "inline";
}

const resetForm = () => {
    let uploadBtn = document.getElementById("submit") as HTMLButtonElement;
    uploadBtn.disabled = false;
    uploadBtn.value = "Upload";

    let reset = document.getElementById("reset") as HTMLButtonElement;
    reset.style.display = "none";

    let detailsDiv = document.getElementById("upload-details-div") as HTMLDivElement;
    detailsDiv.style.display = "inline";

    setFormEnabled(true);
}

/**
 * Parses the HTMLFormElement fields into a SendForm struct
 */
const getFormValues = (): SendForm => {
    let files = (document.getElementById("upload") as HTMLInputElement).files;
    let pw = (document.getElementById("password") as HTMLInputElement).value;
    let pwConfirm = (document.getElementById("confirm-password") as HTMLInputElement).value;
    let downloads = (document.getElementById("downloads") as HTMLInputElement).value;
    let exp = (document.getElementById("expiration") as HTMLInputElement).value;
    let unit = indexToExpUnit((document.getElementById("duration-unit") as HTMLSelectElement).selectedIndex);
    let text = (document.getElementById("upload-text-content") as HTMLTextAreaElement).value;

    // If the password checkbox isn't checked, unset password
    let usePassword = (document.getElementById("use-password") as HTMLInputElement).checked;
    if (!usePassword) {
        pw = pwConfirm = "";
    }

    return {
        files: files,
        password: pw,
        passwordConfirm: pwConfirm,
        downloads: downloads ? parseInt(downloads) : 0,
        expiration: exp ? parseInt(exp) : 0,
        expUnits: unit,
        text: text,
    };
}

const validateForm = (form: SendForm) => {
    let files = form.files;

    if (isFileUpload() && (!files || files.length === 0)) {
        alert("Select at least one file to upload");
        return false;
    }

    if (!validatePassword(form.password, form.passwordConfirm)) {
        alert("Passwords don't match");
        return false;
    }

    if (!validateExpiration(form.expiration, form.expUnits)) {
        return false;
    }

    if (!validateDownloads(form.downloads)) {
        return false;
    }

    // All fields have been validated
    return true;
}

/**
 * Updates the "send used" vs "send available" progress bar and the remaining amount
 * indicator below the bar
 * @param newAmount {number} - The number of bytes added after upload (optional)
 */
const updateProgressBar = (newAmount: number = 0) => {
    let progressBar = document.getElementById("send-bar") as HTMLProgressElement;
    if (!progressBar) {
	    return;
    }

    let sendRemainingSpan = document.getElementById("send-remaining") as HTMLSpanElement;

    if (newAmount > 0) {
        progressBar.value += newAmount;
    }

    let used = calcFileSize(progressBar.value);
    let max = calcFileSize(progressBar.max);
    let remaining = calcFileSize(progressBar.max - progressBar.value);
    sendRemainingSpan.innerText = `${used} / ${max} (${remaining} remaining)`;
}

/**
 * Submits a multi-file form, zipping the contents together and then encrypting
 * the zip file.
 * @param form {SendForm} - The form values being submitted
 * @param key {CryptoKey} - The key to use for encrypting the zip file
 * @param salt {Uint8Array} - The salt used when generating the key
 * @param callback {function()} - The callback function indicating success
 */
const submitFormMulti = async (
    form: SendForm,
    key: CryptoKey,
    salt: Uint8Array,
    callback: () => void,
) => {
    let nameField = document.getElementById("name") as HTMLInputElement;
    let name = nameField.value || "download.zip";
    if (name.endsWith(".zip.zip")) {
        name = name.replace(".zip.zip", ".zip");
    } else if (!name.endsWith(".zip")) {
        name = name + ".zip";
    }

    // @ts-ignore
    let zip = JSZip();
    let size = 0;

    for (let i = 0; i < form.files.length; i++) {
        let file = form.files[i];

        if (file.webkitRelativePath) {
            // @ts-ignore
            zip.file(file.webkitRelativePath, file);
        } else {
            // @ts-ignore
            zip.file(file.name, file);
        }

        size += file.size;
    }

    let encryptedName = await crypto.encryptString(key, name);

    let hexName = toHexString(encryptedName);
    let chunks = getNumChunks(size);
    let expString = getExpString(form.expiration, form.expUnits);

    updateProgress("Uploading file...");
    transfer.uploadSendMetadata(new interfaces.UploadMetadata({
        name: hexName,
        chunks: chunks,
        salt: Array.from(salt),
        downloads: form.downloads,
        size: size,
        expiration: expString
    }), (id) => {
        uploadZip(id, key, zip, chunks).then(() => {
            callback();
        });
    }, () => {
        alert("Failed to upload metadata");
    });
}

/**
 * Submit a single file to YeetFile Send
 * @param form {SendForm} - The send form that is being submitted
 * @param key {CryptoKey} - The randomly generated file key
 * @param secret {string} - The file's key (if no password specified) or salt
 * @param callback {function()} - The function to run after finishing
 */
const submitFormSingle = async (
    form: SendForm,
    key: CryptoKey,
    secret: string,
    callback: () => void,
) => {
    let file = form.files[0];
    let encryptedName = await crypto.encryptString(key, file.name);

    let hexName = toHexString(encryptedName);
    let chunks = getNumChunks(file.size);
    let expString = getExpString(form.expiration, form.expUnits);

    transfer.uploadSendMetadata(new interfaces.UploadMetadata({
        name: hexName,
        chunks: chunks,
        salt: [],
        downloads: form.downloads,
        size: file.size,
        expiration: expString
    }), (id) => {
        let chunk = 1;
        let percent = (chunk / chunks) * 100;
        transfer.uploadSendChunks(id, file, key, (done: boolean) => {
            if (done) {
                showFileTag(id, secret);
                updateProgressBar(file.size);
            } else {
                updateProgress(`Uploading... (${percent}%)`);
            }
            callback();
        }, err => {
            resetForm();
            console.error(err);
        });
    }, () => {
        resetForm();
        console.error("Failed to upload metadata");
    });
}

/**
 * Submits the text-only upload form
 * @param form {HTMLFormElement} - The form being submitted
 * @param key {CryptoKey} - The key used for encrypting the text
 * @param secret {string} - The hex key (if no password was set) otherwise the
 * hex salt.
 * @param callback {function()} - The callback indicating upload status
 */
const submitFormText = async (
    form: SendForm,
    key: CryptoKey,
    secret: string,
    callback: () => void,
) => {
    let encryptedText = await crypto.encryptString(key, form.text);
    let encryptedName = await crypto.encryptString(key, genRandomString(10));

    let hexName = toHexString(encryptedName);
    let expString = getExpString(form.expiration, form.expUnits);
    let downloads = form.downloads;

    uploadTextOnly(hexName, encryptedText, new Uint8Array(), downloads, expString, (tag) => {
        if (tag) {
            showFileTag(tag, secret);
            callback();
        } else {
            resetForm();
        }
    });
}

const uploadZip = async (id, key, zip, chunks) => {
    let i = 0;
    let zipData = new Uint8Array(0);

    zip.generateInternalStream({type:"uint8array"}).on("data", async (data: Uint8Array) => {
        zipData = concatTypedArrays(zipData, data);
        if (zipData.length >= chunkSize) {
            let slice = zipData.subarray(0, chunkSize);
            let blob = await crypto.encryptChunk(key, slice);

            updateProgress(`Uploading file... ${i + 1}/${chunks}`)
            transfer.sendChunk(
                Endpoints.UploadSendFileData,
                blob,
                id,
                i + 1,
                () => {
                    //
                },
                () => {
                    alert("Error uploading file!");
                });
            zipData = zipData.subarray(chunkSize, zipData.length);
            i += 1;
        }
    }).on("end", async () => {
        if (zipData.length > 0) {
            let blob = await crypto.encryptChunk(key, zipData);
            updateProgress(`Uploading file... ${i + 1}/${chunks}`);
            transfer.sendChunk(Endpoints.UploadSendFileData, blob, id, i + 1, (tag) => {
                showFileTag(tag, "");
            }, () => {
                alert("Error uploading file!");
            });
        }
    }).resume();
}

/**
 * Uploads text (not a file) to YeetFile Send
 * @param name {string} - The pseudo-name for the text (not shown to recipient)
 * @param text {Uint8Array} - The encrypted text content
 * @param salt {Uint8Array} - The salt used when encrypting the text
 * @param downloads {number} - The number of possible downloads
 * @param exp {string} - The expiration string
 * @param callback {function(string)} - The function indicating upload completion
 */
const uploadTextOnly = (
    name: string,
    text: Uint8Array,
    salt: Uint8Array,
    downloads: number,
    exp: string,
    callback: (string) => void) => {
    let xhr = new XMLHttpRequest();
    xhr.open("POST", Endpoints.UploadSendText.path, false);
    xhr.setRequestHeader('Content-Type', 'application/json');

    xhr.onreadystatechange = () => {
        if (xhr.readyState === 4 && xhr.status === 200) {
            let response = new interfaces.MetadataUploadResponse(xhr.responseText);
            callback(response.id);
        } else if (xhr.readyState === 4 && xhr.status !== 200) {
            alert(`Error ${xhr.status}: ${xhr.responseText}`);
            callback("");
        }
    };

    xhr.send(JSON.stringify({
        name: name,
        salt: Array.from(salt),
        downloads: downloads,
        expiration: exp,
        text: Array.from(text),
        size: text.length,
    }));
}

const validatePassword = (pwInput, pwConfirm) => {
    return (pwInput.length === 0 || pwConfirm === pwInput);
}

const validateDownloads = (numDownloads) => {
    let maxDownloads = 10;
    if (numDownloads > maxDownloads) {
        alert(`The number of downloads must be between 0-${maxDownloads}.`);
        return false;
    }

    return true;
}

const showFileTag = (id, secret) => {
    let tagDiv = document.getElementById("file-tag-div");
    let fileLink = document.getElementById("file-link") as HTMLAnchorElement;

    let endpoint = Endpoints.format(Endpoints.HTMLSendDownload, id);
    let link = `${window.location.protocol}//${window.location.host}${endpoint}#${secret}`;

    if (window.innerWidth <= 425) {
        let linkLabel = document.getElementById("link-label") as HTMLSpanElement;
        linkLabel.style.fontWeight = "bold";

        let tagRows = document.getElementsByClassName("tag-row");
        Array.from(tagRows).map((tagRow: HTMLTableRowElement) => {
            tagRow.style.display = "none";
        });

        let detailsDiv = document.getElementById("upload-details-div") as HTMLDivElement;
        detailsDiv.style.display = "none";
    }

    tagDiv.style.display = "inherit";
    fileLink.textContent = link;
    fileLink.href = link;
}

const setupCopyButton = () => {
    let copyBtn = document.getElementById("copy-link") as HTMLButtonElement;
    copyBtn.addEventListener("click", event => {
        event.preventDefault();
        let link = document.getElementById("file-link") as HTMLAnchorElement;
        copyToClipboard(link.href, success => {
            if (success) {
                let originalLabel = copyBtn.innerText;
                copyBtn.innerText = "Copied to clipboard!"
                setTimeout(() => {
                    copyBtn.innerText = originalLabel;
                }, 2000);
            } else {
                alert("Failed to copy link to clipboard");
            }
        });
    })
}

const setupInsecureLinkButton = () => {
    let insecureLinkBtn = document.getElementById("create-insecure-link") as HTMLButtonElement;
    if (!insecureLinkBtn) {
        return;
    }

    let link = document.getElementById("file-link") as HTMLAnchorElement;
    insecureLinkBtn.addEventListener("click", event => {
        event.preventDefault();

        let linkArray = link.href.split("#");
        if (linkArray.length <= 1) {
            alert("Invalid link format");
            return;
        }

        let baseLink = linkArray.slice(0, linkArray.length - 1);
        let secret = linkArray[linkArray.length - 1];

        let newLink = `${baseLink}?secret=${secret}`;
        link.href = newLink;
        link.innerText = newLink;

        insecureLinkBtn.className = "hidden";
    });
}

const setupTypeToggles = () => {
    let uploadTextBtn = document.getElementById("upload-text-btn");
    let uploadTextRow = document.getElementById("upload-text-row");

    let uploadFileBtn = document.getElementById("upload-file-btn");
    let uploadFileRow = document.getElementById("upload-file-row");

    uploadTextBtn.addEventListener("click", () => {
        uploadTextRow.style.display = "contents";
        uploadFileRow.style.display = "none";
    });

    uploadFileBtn.addEventListener("click", () => {
        uploadTextRow.style.display = "none";
        uploadFileRow.style.display = "contents";
    });
}

const isFileUpload = () => {
    let uploadFileBtn = document.getElementById("upload-file-btn") as HTMLInputElement;
    return uploadFileBtn.checked;
}

if (document.readyState !== "loading") {
    init();
} else {
    document.addEventListener("DOMContentLoaded", () => {
        init();
    });
}
