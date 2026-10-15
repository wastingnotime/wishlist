import { appPath } from "../lib/paths";

export const identityPath = (file: string) => appPath(`/identity/v1.0.0/${file}`);

export function WishlistBrand() {
  return <a class="brand" href={appPath("/")} aria-label="Wasting No Time Wishlist home">
    <img class="brand-mark" src={identityPath("mark-green.svg")} width="32" height="24" alt="" />
    <span>Wishlist</span>
  </a>;
}

export function MakerSignature() {
  return <a class="maker-signature" href="https://wastingnotime.org/" aria-label="made by WNT — Wasting No Time home">
    <img src={identityPath("mark-green.svg")} width="24" height="18" alt="" />
    <span>made by WNT</span>
  </a>;
}
