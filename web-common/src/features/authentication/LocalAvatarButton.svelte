<script lang="ts">
  import { page } from "$app/stores";
  import Avatar from "@rilldata/web-common/components/avatar/Avatar.svelte";
  import * as DropdownMenu from "@rilldata/web-common/components/dropdown-menu";
  import NoUser from "@rilldata/web-common/components/icons/NoUser.svelte";
  import { EntityStatus } from "@rilldata/web-common/features/entity-management/types";
  import { initPylonChat } from "@rilldata/web-common/features/help/initPylonChat";
  import {
    createLocalServiceGetCurrentUser,
    createLocalServiceGetMetadata,
  } from "@rilldata/web-common/runtime-client/local-service";
  import Spinner from "@rilldata/web-common/features/entity-management/Spinner.svelte";
  import ThemeToggle from "@rilldata/web-common/features/themes/ThemeToggle.svelte";

  export let externalUser: { email: string; name: string } | null = null;
  export let onLogout: (() => void) | null = null;
  // Optional Bratrax onboarding-checklist link (e.g. "Continue setup"). Passed
  // down from the app layout, which knows the user's role; null hides it.
  export let onboardingLink: { href: string; label: string } | null = null;

  $: user = createLocalServiceGetCurrentUser({
    query: {
      // refetch in case user does a login/logout from outside of rill developer UI
      refetchOnWindowFocus: true,
    },
  });
  $: metadata = createLocalServiceGetMetadata();

  let loginUrl: string;
  $: if ($metadata.data?.loginUrl) {
    const u = new URL($metadata.data.loginUrl);
    u.searchParams.set(
      "redirect",
      `${window.location.origin}${window.location.pathname}`,
    );
    loginUrl = u.toString();
  }

  let logoutUrl: string;
  $: if ($metadata.data?.loginUrl) {
    const u = new URL($metadata.data.loginUrl + "/logout");
    u.searchParams.set("redirect", $page.url.href);
    logoutUrl = u.toString();
  }

  $: rillUser = $user.isSuccess && $user.data?.user;
  $: loggedIn = externalUser || rillUser;

  $: if ($user.data?.user) {
    initPylonChat($user.data.user);
  }
  function handlePylon() {
    window.Pylon("show");
  }

  $: initials = externalUser
    ? (externalUser.name || externalUser.email || "?")
        .split(/[\s@]/)
        .filter(Boolean)
        .slice(0, 2)
        .map((s) => s[0].toUpperCase())
        .join("")
    : "";

  let photoUrlErrored = false;
</script>

{#if !externalUser && ($user.isLoading || $metadata.isLoading) && !$user.error && !$metadata.error}
  <div class="flex flex-row items-center h-7 mx-1.5">
    <Spinner size="16px" status={EntityStatus.Running} />
  </div>
{:else}
  <DropdownMenu.Root>
    <DropdownMenu.Trigger
      class="flex-none w-7"
      aria-label="Avatar logged {loggedIn ? 'in' : 'out'}"
    >
      {#if externalUser}
        <div
          class="h-7 w-7 rounded-full bg-primary-500 text-white flex items-center justify-center text-xs font-medium"
        >
          {initials}
        </div>
      {:else if rillUser && !photoUrlErrored && $user.data && $metadata.data}
        <Avatar
          src={$user.data?.user?.photoUrl}
          alt={$user.data?.user?.displayName || $user.data?.user?.email}
          avatarSize="h-7 w-7"
        />
      {:else}
        <NoUser />
      {/if}
    </DropdownMenu.Trigger>
    <DropdownMenu.Content class="p-1">
      {#if externalUser}
        <div class="px-3 py-1.5 text-xs text-gray-500">
          {externalUser.email}
        </div>
        <DropdownMenu.Separator />
      {/if}

      {#if externalUser && onboardingLink}
        <DropdownMenu.Item href={onboardingLink.href}>
          {onboardingLink.label}
        </DropdownMenu.Item>
        <DropdownMenu.Separator />
      {/if}

      <ThemeToggle />

      {#if !externalUser}
        <DropdownMenu.Separator />
        <DropdownMenu.Item
          href="https://docs.rilldata.com"
          target="_blank"
          rel="noreferrer noopener"
        >
          Documentation
        </DropdownMenu.Item>
      {/if}

      {#if externalUser && onLogout}
        <DropdownMenu.Separator />
        <DropdownMenu.Item on:click={onLogout}>
          Logout
        </DropdownMenu.Item>
      {:else if rillUser}
        <DropdownMenu.Item on:click={handlePylon}>
          Contact Bratrax support
        </DropdownMenu.Item>
        <DropdownMenu.Separator />
        <DropdownMenu.Item href={logoutUrl} rel="external">
          Logout
        </DropdownMenu.Item>
      {:else if !externalUser}
        <DropdownMenu.Separator />
        <DropdownMenu.Item href={loginUrl} rel="external">
          Log in / Sign up
        </DropdownMenu.Item>
      {/if}
    </DropdownMenu.Content>
  </DropdownMenu.Root>
{/if}
