/**
 * Shipping regions a per-order rate can be overridden for.
 *
 * Codes are fully qualified — "US-AK", not "AK" — because province and country
 * codes share one namespace and collide otherwise: California and Canada are
 * both "CA". Shopify reports Puerto Rico as a country with no province, so it
 * keys on "PR" alone. The warehouse builds the same key per order in
 * Order.computed_shipping_cost (clients/<name>/ontology.yaml).
 */
export interface ShippingRegion {
  code: string;
  label: string;
}

/** Offered first: the regions that routinely cost more than the lower 48. */
export const SUGGESTED_REGION_CODES = ["US-AK", "US-HI", "PR"];

export const SHIPPING_REGIONS: ShippingRegion[] = [
  { code: "US-AL", label: "Alabama" },
  { code: "US-AK", label: "Alaska" },
  { code: "US-AZ", label: "Arizona" },
  { code: "US-AR", label: "Arkansas" },
  { code: "US-CA", label: "California" },
  { code: "US-CO", label: "Colorado" },
  { code: "US-CT", label: "Connecticut" },
  { code: "US-DE", label: "Delaware" },
  { code: "US-DC", label: "District of Columbia" },
  { code: "US-FL", label: "Florida" },
  { code: "US-GA", label: "Georgia" },
  { code: "US-HI", label: "Hawaii" },
  { code: "US-ID", label: "Idaho" },
  { code: "US-IL", label: "Illinois" },
  { code: "US-IN", label: "Indiana" },
  { code: "US-IA", label: "Iowa" },
  { code: "US-KS", label: "Kansas" },
  { code: "US-KY", label: "Kentucky" },
  { code: "US-LA", label: "Louisiana" },
  { code: "US-ME", label: "Maine" },
  { code: "US-MD", label: "Maryland" },
  { code: "US-MA", label: "Massachusetts" },
  { code: "US-MI", label: "Michigan" },
  { code: "US-MN", label: "Minnesota" },
  { code: "US-MS", label: "Mississippi" },
  { code: "US-MO", label: "Missouri" },
  { code: "US-MT", label: "Montana" },
  { code: "US-NE", label: "Nebraska" },
  { code: "US-NV", label: "Nevada" },
  { code: "US-NH", label: "New Hampshire" },
  { code: "US-NJ", label: "New Jersey" },
  { code: "US-NM", label: "New Mexico" },
  { code: "US-NY", label: "New York" },
  { code: "US-NC", label: "North Carolina" },
  { code: "US-ND", label: "North Dakota" },
  { code: "US-OH", label: "Ohio" },
  { code: "US-OK", label: "Oklahoma" },
  { code: "US-OR", label: "Oregon" },
  { code: "US-PA", label: "Pennsylvania" },
  { code: "US-RI", label: "Rhode Island" },
  { code: "US-SC", label: "South Carolina" },
  { code: "US-SD", label: "South Dakota" },
  { code: "US-TN", label: "Tennessee" },
  { code: "US-TX", label: "Texas" },
  { code: "US-UT", label: "Utah" },
  { code: "US-VT", label: "Vermont" },
  { code: "US-VA", label: "Virginia" },
  { code: "US-WA", label: "Washington" },
  { code: "US-WV", label: "West Virginia" },
  { code: "US-WI", label: "Wisconsin" },
  { code: "US-WY", label: "Wyoming" },
  // US territories Shopify reports as their own country, hence no "US-" prefix.
  { code: "PR", label: "Puerto Rico" },
  { code: "VI", label: "U.S. Virgin Islands" },
  { code: "GU", label: "Guam" },
  { code: "AS", label: "American Samoa" },
  { code: "MP", label: "Northern Mariana Islands" },
];

const LABELS = new Map(SHIPPING_REGIONS.map((r) => [r.code, r.label]));

/** Falls back to the raw code so a region saved elsewhere still renders. */
export function regionLabel(code: string): string {
  return LABELS.get(code) ?? code;
}
