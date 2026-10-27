/** Marca On-Wallet: ícone + wordmark tipográfico (hífen amarelo da identidade). */
export default function BrandLogo({
  size = 32,
  short = false,
  stacked = false,
}: {
  size?: number;
  /** Só o ícone (mobile / favicon visual). */
  short?: boolean;
  /** Ícone acima do nome — login / hero. */
  stacked?: boolean;
}) {
  const name = (
    <span className="brand-wordmark" aria-hidden={short ? true : undefined}>
      On<span className="brand-hyphen">-</span>Wallet
    </span>
  );

  if (stacked) {
    return (
      <div className="brand-stack">
        <img src="/on-wallet-mark.png" alt="" width={size * 2.2} height={size * 2.2} className="brand-mark" />
        <span className="brand-wordmark brand-wordmark-lg">
          On<span className="brand-hyphen">-</span>Wallet
        </span>
      </div>
    );
  }

  return (
    <>
      <img src="/on-wallet-mark.png" alt="" width={size} height={size} className="brand-mark" />
      {!short && name}
      {short && <span className="visually-hidden">On-Wallet</span>}
    </>
  );
}
