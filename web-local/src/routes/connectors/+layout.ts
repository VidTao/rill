export const ssr = false;

export async function load() {
  const { fetchAllConnections } = await import(
    "$lib/bratrax/connectors/connection-store"
  );
  void fetchAllConnections();
  return {};
}
