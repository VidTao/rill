import { writable } from "svelte/store";
import type {
  AdvertisingConnection,
  CrmConnection,
  PlatformConnection,
} from "./types";
import {
  getAllPlatformConnections,
  getAllAdConnections,
  getAllCrmConnections,
} from "./api";

export const platformConnections = writable<
  Record<string, PlatformConnection>
>({});

export const adConnections = writable<
  Record<string, AdvertisingConnection[]>
>({});

export const crmConnections = writable<Record<string, CrmConnection[]>>({});

export const connectionsLoading = writable<boolean>(false);

export async function fetchAllConnections(): Promise<void> {
  connectionsLoading.set(true);
  try {
    const [platforms, ads, crms] = await Promise.all([
      getAllPlatformConnections().catch(() => ({})),
      getAllAdConnections().catch(() => ({})),
      getAllCrmConnections().catch(() => ({})),
    ]);
    platformConnections.set(platforms);
    adConnections.set(ads);
    crmConnections.set(crms);
  } finally {
    connectionsLoading.set(false);
  }
}
