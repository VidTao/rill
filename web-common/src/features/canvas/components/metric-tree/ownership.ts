export type OwnershipNode = {
  id: string;
  label: string;
  type: string;
  owner?: string | null;
  ownerUserId?: number | null;
  ownerLabel?: string | null;
  status?: string | null;
};

export type OwnershipPersona = {
  owner: string;
  ownerKey: string;
  ownerUserId?: number | null;
  leverCount: number;
  experimentCount: number;
  runningCount: number;
  backlogCount: number;
  unassigned: boolean;
  nodes: Array<{ id: string; label: string; type: string; status?: string | null }>;
};

export function ownerKeyForNode(node: OwnershipNode | null | undefined): string {
  if (!node) return "unassigned";
  if (node.ownerUserId != null) return `user:${node.ownerUserId}`;
  const label = (node.ownerLabel || node.owner || "").trim();
  return label ? `label:${label.toLowerCase()}` : "unassigned";
}

export function ownerLabelForNode(node: OwnershipNode | null | undefined): string {
  if (!node) return "Unassigned";
  return (node.ownerLabel || node.owner || "").trim() || "Unassigned";
}

export function ownershipRosterFor(nodes: OwnershipNode[] | null | undefined): OwnershipPersona[] {
  if (!nodes?.length) return [];
  const groups = new Map<string, OwnershipPersona>();
  for (const node of nodes) {
    if (node.type !== "lever" && node.type !== "experiment") continue;
    const ownerKey = ownerKeyForNode(node);
    const owner = ownerLabelForNode(node);
    const existing = groups.get(ownerKey) ?? {
      owner,
      ownerKey,
      ownerUserId: node.ownerUserId ?? null,
      leverCount: 0,
      experimentCount: 0,
      runningCount: 0,
      backlogCount: 0,
      unassigned: owner === "Unassigned",
      nodes: [],
    };
    if (node.type === "lever") existing.leverCount += 1;
    if (node.type === "experiment") {
      existing.experimentCount += 1;
      if (node.status === "running") existing.runningCount += 1;
      if (!node.status || node.status === "backlog" || node.status === "do_now") {
        existing.backlogCount += 1;
      }
    }
    existing.nodes.push({
      id: node.id,
      label: node.label,
      type: node.type,
      status: node.status,
    });
    groups.set(ownerKey, existing);
  }
  return [...groups.values()].sort((a, b) => {
    if (a.unassigned !== b.unassigned) return a.unassigned ? -1 : 1;
    const aWeight = a.leverCount * 10 + a.runningCount * 4 + a.experimentCount;
    const bWeight = b.leverCount * 10 + b.runningCount * 4 + b.experimentCount;
    return bWeight - aWeight || a.owner.localeCompare(b.owner);
  });
}

export function ownerCoverageLabel(roster: OwnershipPersona[]): string {
  const unassigned = roster.find((owner) => owner.unassigned);
  if (!roster.length) return "No owned levers or experiments";
  if (unassigned) {
    const count = unassigned.leverCount + unassigned.experimentCount;
    return `${count} unassigned action item${count === 1 ? "" : "s"}`;
  }
  return `${roster.length} owner${roster.length === 1 ? "" : "s"} accountable`;
}

export function ownerNodeText(item: OwnershipPersona["nodes"][number]): string {
  return [item.type, item.status].filter(Boolean).join(" · ");
}

export function ownerMatchesNode(owner: string | null | undefined, node: OwnershipNode | null | undefined): boolean {
  if (!owner) return true;
  return ownerKeyForNode(node) === owner || ownerLabelForNode(node) === owner;
}
