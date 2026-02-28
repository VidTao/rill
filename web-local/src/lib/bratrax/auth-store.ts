import { writable } from "svelte/store";
import type { BratraxUser } from "./auth";

export const bratraxUser = writable<BratraxUser | null>(null);
export const bratraxAuthChecked = writable<boolean>(false);
