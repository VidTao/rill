<script>
  export let size = "16px";
  export let color = "#374151";
</script>

<svg
  xmlns="http://www.w3.org/2000/svg"
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  stroke={color}
  stroke-width="2"
  stroke-linecap="round"
  stroke-linejoin="round"
>
  <!-- parent node -->
  <rect x="9" y="2" width="6" height="4" rx="1" />
  <!-- two child nodes -->
  <rect x="2" y="18" width="6" height="4" rx="1" />
  <rect x="16" y="18" width="6" height="4" rx="1" />
  <!-- connectors: parent down to a branch point, then out to each child -->
  <path d="M12 6v5" />
  <path d="M5 18v-3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v3" />
  <path d="M19 14v0" />
</svg>
