import * as transfer from "./transfer.js";

export const DialogSignal = {
    Cancel: Symbol("cancel"),
    Rename: Symbol("rename"),
    Share: Symbol("share"),
}

/**
 *
 * @param id {string}
 * @param rawKey {ArrayBuffer}
 * @param isFolder {boolean}
 * @param callback {function(Symbol)}
 */
export const showShareDialog = (id, rawKey, isFolder, callback) => {
    let shareDialog = document.getElementById("share-dialog");
    let shareTarget = document.getElementById("share-target");
    let shareModify = document.getElementById("share-modify");

    let submitBtn = document.getElementById("submit-share");
    let cancelBtn = document.getElementById("cancel-share");

    let shareLoading = document.getElementById("share-loading");
    let shareTable = document.getElementById("share-table");
    let shareTableBody = document.getElementById("share-table-body");

    shareLoading.style.display = "inherit";
    shareTable.style.display = "none";
    shareTableBody.innerHTML = "";

    transfer.getSharedUsers(id, isFolder).then(response => {
        shareLoading.style.display = "none";
        if (response && response.length !== 0) {
            shareTable.style.display = "inherit";
        } else {
            return;
        }

        for (let i = 0; i < response.length; i++) {
            generateShareRow(id, shareTableBody, response[i], isFolder, callback);
        }
    });

    submitBtn.addEventListener("click", async event => {
        event.stopPropagation();

        if (shareTarget.value.length === 0) {
            alert("Must enter a YeetFile user's email or account ID for sharing");
            return;
        }

        updateButton(submitBtn, true, "Sharing...");

        let target = shareTarget.value;
        let canModify = shareModify.checked;
        transfer.shareItem(target, rawKey, id, canModify, isFolder).then(response => {
            let name = target;
            if (!target.includes("@")) {
                name = "*" + name.substring(name.length - 4, name.length);
            }
            generateShareRow(id, shareTableBody, {id: response.id, recipientName: name}, isFolder, callback);
            updateButton(submitBtn, false, "Share");
            callback(DialogSignal.Share);
        }).catch(() => {
            updateButton(submitBtn, false, "Share");
        });
    });

    cancelBtn.addEventListener("click", event => {
        event.stopPropagation();
        closeDialog(shareDialog);
        callback(DialogSignal.Cancel);
    })

    shareDialog.showModal();
}

const generateShareRow = (id, tableBody, recipient, isFolder, callback) => {
    let row = `<tr id="share-${recipient.id}">
<td>${recipient.recipientName}</td>
<td><input id="can-modify-${recipient.id}" type="checkbox" ${recipient.canModify ? "checked" : ""}></td>
<td><img id="delete-modify-${recipient.id}" class="vault-icon red-icon" src="/static/icons/remove.svg"></td>
</tr>`;

    tableBody.parentElement.style.display = "table";

    tableBody.innerHTML += row;
    tableBody.addEventListener("click", event => {
        if (event.target.id === `can-modify-${recipient.id}`) {
            let cb = document.getElementById(`can-modify-${recipient.id}`);
            transfer.changeSharedItemPerms(id, recipient.id, cb.checked, isFolder);
        } else if (event.target.id === `delete-modify-${recipient.id}`) {
            if (confirm(`Remove user '${recipient.recipientName}' from shared content?`)) {
                transfer.removeUserFromShared(id, recipient.id, isFolder).then(() => {
                    callback(DialogSignal.Share);

                    document.getElementById(`share-${recipient.id}`).remove();
                    // Hide table if there aren't any users left
                    if (tableBody.children.length === 0) {
                        tableBody.parentElement.style.display = "none";
                    }
                });
            }

        }
    });
}

export const closeDialogs = () => {
    let dialogs = document.getElementsByTagName("dialog");
    for (let i = 0; i < dialogs.length; i++) {
        closeDialog(dialogs[i]);
    }
}

export const closeDialog = (dialog) => {
    dialog.close();

    // Reset listeners for dynamic dialogs
    if (dialog.dataset.dynamic) {
        dialog.outerHTML = dialog.outerHTML;
    }
}

export const isDialogOpen = () => {
    let dialogs = document.getElementsByTagName("dialog");
    for (let i = 0; i < dialogs.length; i++) {
        if (!dialogs[i].open) {
            continue;
        }

        return true;
    }

    return false;
}