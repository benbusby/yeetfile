const fetchLocalStorageString = (key: string, fallback?: string): string => {
    let value = localStorage.getItem(key);
    if (!value && fallback !== undefined) {
        return fallback;
    } else if (!value) {
        return "";
    }

    return value;
}

const fetchLocalStorageInt = (key: string, fallback?: number): number => {
    let value = fetchLocalStorageString(key);
    if (!value && fallback !== undefined) {
        return fallback;
    } else if (!value) {
        return 0;
    }

    let numValue = parseInt(value);
    if (isNaN(numValue) && fallback !== undefined) {
        return fallback;
    } else if (isNaN(numValue)) {
        return 0;
    }

    return numValue;
}

// =============================================================================
// Vault Password Settings
// =============================================================================

const useVaultPasswordKey = "UseVaultPassword";
const useVaultPasswordValue = "true";

export const getVaultPasswordSetting = (): boolean => {
    return fetchLocalStorageString(useVaultPasswordKey) === useVaultPasswordValue;
}

export const enableVaultPasswordSetting = () => {
    localStorage.setItem(useVaultPasswordKey, useVaultPasswordValue);
}

export const disableVaultPasswordSetting = () => {
    localStorage.setItem(useVaultPasswordKey, "");
}

// =============================================================================
// Default Settings (YeetFile Send)
// =============================================================================

const defaultSendDownloadsKey = "DefaultSendDownloads";
const defaultSendExpirationKey = "DefaultSendExpiration";
const defaultSendExpirationUnitsKey = "DefaultSendExpirationUnits";

const defaultSendDownloads = 1;
const defaultSendExpiration = 30;
const defaultSendExpirationUnits = 0; // ExpUnits.Minutes

export const getDefaultSendDownloads = (): number => {
    return fetchLocalStorageInt(
        defaultSendDownloadsKey,
        defaultSendDownloads);
}

export const getDefaultSendExpiration = (): number => {
    return fetchLocalStorageInt(
        defaultSendExpirationKey,
        defaultSendExpiration);
}

export const getDefaultSendExpirationUnits = (): ExpUnits => {
    return fetchLocalStorageInt(
        defaultSendExpirationUnitsKey,
        defaultSendExpirationUnits);
}

export const setDefaultSendValues = (
    downloads: number,
    expiration: number,
    units: ExpUnits,
) => {
    localStorage.setItem(defaultSendDownloadsKey, String(downloads));
    localStorage.setItem(defaultSendExpirationKey, String(expiration));
    localStorage.setItem(defaultSendExpirationUnitsKey, String(units.valueOf()));
}