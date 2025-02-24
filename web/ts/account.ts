import {Endpoints} from "./endpoints.js";
import {YeetFileDB} from "./db.js";
import * as interfaces from "./interfaces.js";
import * as localstorage from "./localstorage.js";

const init = () => {
    loadStoredSettings();

    let logoutBtn = document.getElementById("logout-btn");
    logoutBtn.addEventListener("click", logout);

    let deleteBtn = document.getElementById("delete-btn");
    deleteBtn.addEventListener("click", deleteAccount);

    let changePwBtn = document.getElementById("change-pw-btn");
    changePwBtn.addEventListener("click", changePassword);

    let changeEmailLink = document.getElementById("change-email");
    if (changeEmailLink) {
        changeEmailLink.addEventListener("click", changeEmail);
    }

    let setEmailLink = document.getElementById("set-email");
    if (setEmailLink) {
        setEmailLink.addEventListener("click", setEmail);
    }

    let disable2FALink = document.getElementById("disable-2fa");
    if (disable2FALink) {
        disable2FALink.addEventListener("click", disable2FA);
    }

    let recyclePaymentIDBtn = document.getElementById("recycle-payment-id");
    recyclePaymentIDBtn.addEventListener("click", recyclePaymentID);

    let yearlyToggle = document.getElementById("yearly-toggle");
    if (yearlyToggle) {
        yearlyToggle.addEventListener("click", () => {
            window.location.assign("/account?yearly=1");
        });
    }

    let monthlyToggle = document.getElementById("monthly-toggle");
    if (monthlyToggle) {
        monthlyToggle.addEventListener("click", () => {
            window.location.assign("/account");
        });
    }

    let saveSettingsBtn = document.getElementById("save-settings-btn") as HTMLButtonElement;
    saveSettingsBtn.addEventListener("click", saveSettings);
}

const loadStoredSettings = () => {
    let defaultDownloads = document.getElementById("send-downloads") as HTMLInputElement;
    let defaultExpiration = document.getElementById("send-exp") as HTMLInputElement;
    let expUnits = document.getElementById("send-exp-units") as HTMLSelectElement;

    defaultDownloads.value = String(localstorage.getDefaultSendDownloads());
    defaultExpiration.value = String(localstorage.getDefaultSendExpiration());
    expUnits.selectedIndex = localstorage.getDefaultSendExpirationUnits();
}

const saveSettings = () => {
    let saveSettingsBtn = document.getElementById("save-settings-btn") as HTMLButtonElement;
    let defaultDownloads = document.getElementById("send-downloads") as HTMLInputElement;
    let defaultExpiration = document.getElementById("send-exp") as HTMLInputElement;
    let expUnits = document.getElementById("send-exp-units") as HTMLSelectElement;

    if (isNaN(parseInt(defaultDownloads.value))) {
        alert("Invalid downloads value");
        return;
    }

    if (isNaN(parseInt(defaultExpiration.value))) {
        alert("Invalid expiration value");
        return;
    }

    let downloadsNum = defaultDownloads.valueAsNumber;
    let expirationNum = defaultExpiration.valueAsNumber;
    let expUnitNum = indexToExpUnit(expUnits.selectedIndex);

    if (downloadsNum < 1 || downloadsNum > 10) {
        alert("Downloads must be between 1-10");
        return;
    }

    if (!validateExpiration(expirationNum, indexToExpUnit(expUnitNum))) {
        return;
    }

    localstorage.setDefaultSendValues(
        defaultDownloads.valueAsNumber,
        defaultExpiration.valueAsNumber,
        expUnits.selectedIndex);

    let originalBtnLabel = saveSettingsBtn.innerText;
    saveSettingsBtn.className = "success-btn";
    saveSettingsBtn.innerText = "Saved!";

    setTimeout(() => {
        saveSettingsBtn.innerText = originalBtnLabel;
        saveSettingsBtn.className = "";
    }, 1500);
}

const logout = () => {
    let confirmMsg = "Log out of YeetFile?";
    if (confirm(confirmMsg)) {
        new YeetFileDB().removeKeys(success => {
            if (success) {
                window.location.assign(Endpoints.Logout.path);
            } else {
                alert("Error removing keys");
            }
        });
    }
}

const deleteAccount = () => {
    let confirmMsg = "Are you sure you want to delete your account? This can " +
        "not be undone."
    if (!confirm(confirmMsg)) {
        return;
    }

    let promptMsg = "Enter your login email or account ID -- below to delete " +
        "your account."

    let id = prompt(promptMsg);
    if (id.length > 0) {
        let request = new interfaces.DeleteAccount();
        request.identifier = id;

        fetch(Endpoints.Account.path, {
            method: "DELETE",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(request)
        }).then(async response => {
            if (response.ok) {
                alert("Your account has been permanently yeeted.");
                window.location.assign("/");
            } else {
                let errMsg = await response.text()
                alert("There was an error deleting your account: " + errMsg);
                return;
            }
        }).catch(error => {
            console.error(error);
            alert("There was an error deleting your account!");
        })
    }
}

const changeEmail = () => {
    let changeMsg = "An email will be sent to your current email to initiate " +
        "the process of changing your email. Do you want to continue?"
    if (confirm(changeMsg)) {
        fetch(Endpoints.format(Endpoints.ChangeEmail, ""), {
            method: "POST",
        }).then(async response => {
            if (response.ok) {
                showMessage("Check your email for instructions on how to " +
                    "update your YeetFile email.", false);
            } else {
                showMessage("Error: " + await response.text(), true);
            }
        }).catch(() => {
            alert("Request failed");
        });
    }
}

const setEmail = () => {
    let setMsg = "If you set an email for your account, you will use it to log " +
        "in instead of your account ID. Do you want to continue?"
    if (confirm(setMsg)) {
        fetch(Endpoints.format(Endpoints.ChangeEmail, ""), {
            method: "POST",
        }).then(async response => {
            if (response.ok) {
                let changeResponse = new interfaces.StartEmailChangeResponse(
                    await response.json()
                );

                if (changeResponse.changeID) {
                    window.location.assign(Endpoints.format(
                        Endpoints.HTMLChangeEmail,
                        changeResponse.changeID));
                }
            } else {
                showMessage("Error: " + await response.text(), true);
            }
        }).catch(() => {
            alert("Request failed");
        });
    }
}

const disable2FA = () => {
    let dialog = document.getElementById("two-factor-dialog") as HTMLDialogElement;
    let message = document.getElementById("two-factor-message") as HTMLParagraphElement;
    let codeInput = document.getElementById("two-factor-code") as HTMLInputElement;
    let submit = document.getElementById("submit-2fa") as HTMLButtonElement;
    let cancel = document.getElementById("cancel-2fa") as HTMLButtonElement;

    message.innerHTML = "To disable two-factor authentication, type in your 2FA " +
        "code or a recovery code below."

    codeInput.addEventListener("keydown", (event: KeyboardEvent) => {
        if (event.key === "Enter") {
            submit.click();
        }
    });

    submit.className = "destructive-btn";
    submit.innerHTML = "Disable";
    submit.addEventListener("click",  () => {
        dialog.close();
        fetch(`${Endpoints.TwoFactor.path}?code=${codeInput.value}`, {
            method: "DELETE",
        }).then(response => {
            if (response.ok) {
                alert("Two-factor authentication disabled!");
                window.location.reload();
            } else {
                alert("Failed to disable 2FA -- double check your 2FA code and try again.");
            }
        });
    });

    cancel.addEventListener("click", () => {
        dialog.close();
    });

    dialog.showModal();
}

const recyclePaymentID = () => {
    let confirmMsg = "Are you sure you want to recycle your payment ID? " +
        "This will remove all records of past payments you've made.";

    if (confirm(confirmMsg)) {
        fetch(
            Endpoints.RecyclePaymentID.path, {method: "PUT"}
        ).then(response => {
            if (response.ok) {
                window.location.assign(Endpoints.HTMLAccount.path);
            } else if (response.status === 400) {
                alert("Failed to recycle payment ID -- please ensure your " +
                    "subscription has been canceled before trying again.");
            } else {
                alert("Failed to recycle payment ID");
            }
        }).catch(() => {
            alert("Error recycling payment ID");
        });
    }
}

const changePassword = () => {
    window.location.assign(Endpoints.HTMLChangePassword.path);
}

if (document.readyState !== "loading") {
    init();
} else {
    document.addEventListener("DOMContentLoaded", () => {
        init();
    });
}