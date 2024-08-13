import * as crypto from "./crypto.js";
import * as interfaces from "./interfaces.js";
import * as transfer from "./transfer.js";
import {Endpoints} from "./endpoints.js";

let pepper = "";

const init = () => {
    setupTypeToggles();

    let usePasswordCB = document.getElementById("use-password") as HTMLInputElement;
    let passwordInput = document.getElementById("password") as HTMLInputElement;
    let confirmPasswordInput = document.getElementById("confirm-password") as HTMLInputElement;
    let showPasswordCB = document.getElementById("show-password") as HTMLInputElement;

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

    form.addEventListener("submit", (event) => {
        event.preventDefault();

        let formValues = getFormValues();

        if (validateForm(formValues)) {
            setFormEnabled(false);
            crypto.generatePassphrase(async passphrase => {
                pepper = passphrase;

                updateProgress("Initializing");
                let [key, salt] = await crypto.deriveSendingKey(formValues.pw, undefined, passphrase);

                if (isFileUpload()) {
                    if (formValues.files.length > 1) {
                        await submitFormMulti(formValues, key, salt, allowReset);
                    } else {
                        await submitFormSingle(formValues, key, salt, allowReset);
                    }
                } else {
                    await submitFormText(formValues, key, salt, allowReset);
                }
            });
        }
    });
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

    let reset = document.getElementById("reset");
    reset.style.display = "none";

    setFormEnabled(true);
}

const getFormValues = () => {
    let files = (document.getElementById("upload") as HTMLInputElement).files;
    let pw = (document.getElementById("password") as HTMLInputElement).value;
    let pwConfirm = (document.getElementById("confirm-password") as HTMLInputElement).value;
    let downloads = (document.getElementById("downloads") as HTMLInputElement).value;
    let exp = (document.getElementById("expiration") as HTMLInputElement).value;
    let unit = indexToExpUnit((document.getElementById("duration-unit") as HTMLSelectElement).selectedIndex);
    let plaintext = (document.getElementById("upload-text-content") as HTMLTextAreaElement).value;

    // If the password checkbox isn't checked, unset password
    let usePassword = (document.getElementById("use-password") as HTMLInputElement).checked;
    if (!usePassword) {
        pw = pwConfirm = "";
    }

    return { files, pw, pwConfirm, downloads, exp, unit, plaintext };
}

const validateForm = (form) => {
    let files = form.files;

    if (isFileUpload() && (!files || files.length === 0)) {
        alert("Select at least one file to upload");
        return false;
    }

    if (!validatePassword(form.pw, form.pwConfirm)) {
        alert("Passwords don't match");
        return false;
    }

    if (!validateExpiration(form.exp, form.unit)) {
        return false;
    }

    if (!validateDownloads(form.downloads)) {
        return false;
    }

    // All fields have been validated
    return true;
}

const submitFormMulti = async (form, key, salt, callback) => {
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
            zip.file(file.webkitRelativePath, file);
        } else {
            zip.file(file.name, file);
        }

        size += file.size;
    }

    let encryptedName = await crypto.encryptString(key, name);

    let hexName = toHexString(encryptedName);
    let chunks = getNumChunks(size);
    let expString = getExpString(form.exp, form.unit);

    updateProgress("Uploading file...");
    transfer.uploadSendMetadata(new interfaces.UploadMetadata({
        name: hexName,
        chunks: chunks,
        salt: Array.from(salt),
        downloads: parseInt(form.downloads),
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

const submitFormSingle = async (form, key, salt, callback) => {
    let file = form.files[0];
    let encryptedName = await crypto.encryptString(key, file.name);

    let hexName = toHexString(encryptedName);
    let chunks = getNumChunks(file.size);
    let expString = getExpString(form.exp, form.unit);
    console.log(expString);

    transfer.uploadSendMetadata(new interfaces.UploadMetadata({
        name: hexName,
        chunks: chunks,
        salt: Array.from(salt),
        downloads: parseInt(form.downloads),
        size: file.size,
        expiration: expString
    }), (id) => {
        transfer.uploadSendChunks(id, file, key, () => {
            showFileTag(id);
            callback();
        }, err => {
            alert("Error uploading file");
            console.error(err);
        });
    }, () => {
        alert("Failed to upload metadata");
    });
}

const submitFormText = async (form, key, salt, callback) => {
    let encryptedText = await crypto.encryptString(key, form.plaintext);
    let encryptedName = await crypto.encryptString(key, genRandomString(10));

    let hexName = toHexString(encryptedName);
    let expString = getExpString(form.exp, form.unit);
    let downloads = parseInt(form.downloads);

    uploadPlaintext(hexName, encryptedText, salt, downloads, expString, (tag) => {
        if (tag) {
            showFileTag(tag);
            callback();
        } else {
            resetForm();
        }
    });
}

const uploadZip = async (id, key, zip, chunks) => {
    let i = 0;
    let zipData = new Uint8Array(0);

    zip.generateInternalStream({type:"uint8array"}).on ('data', async (data, metadata) => {
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
                showFileTag(tag);
            }, () => {
                alert("Error uploading file!");
            });
        }
    }).resume();
}

const uploadFileChunks = async (id, key, file, chunks) => {
    for (let i = 0; i < chunks; i++) {
        let start = i * chunkSize;
        let end = (i + 1) * chunkSize;

        if (end > file.size) {
            end = file.size;
        }

        let data = await file.slice(start, end).arrayBuffer();
        let blob = await crypto.encryptChunk(key, new Uint8Array(data));

        updateProgress(`Uploading file... ${i + 1}/${chunks}`)
        transfer.sendChunk(Endpoints.UploadSendFileData, blob, id, i + 1, (tag) => {
            if (tag) {
                showFileTag(tag);
            }
        }, () => {
            alert("Error uploading file!");
        });
    }
}

const uploadPlaintext = (name, text, salt, downloads, exp, callback) => {
    let xhr = new XMLHttpRequest();
    xhr.open("POST", Endpoints.UploadSendText.path, false);
    xhr.setRequestHeader('Content-Type', 'application/json');

    xhr.onreadystatechange = () => {
        if (xhr.readyState === 4 && xhr.status === 200) {
            let response = new interfaces.MetadataUploadResponse(xhr.responseText);
            callback(response.id);
        } else if (xhr.readyState === 4 && xhr.status !== 200) {
            alert(`Error ${xhr.status}: ${xhr.responseText}`);
            callback();
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

const showFileTag = (tag) => {
    let tagDiv = document.getElementById("file-tag-div");
    let fileTag = document.getElementById("file-tag");
    let fileLink = document.getElementById("file-link") as HTMLAnchorElement;

    let link = `${window.location.protocol}//${window.location.host}/send/${tag}#${pepper}`

    tagDiv.style.display = "inherit";
    fileTag.textContent = `${tag}#${pepper}`;
    fileLink.textContent = link;
    fileLink.href = link;
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