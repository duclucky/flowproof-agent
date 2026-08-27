package httpapi

import "net/http"

func demoStore(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(demoStoreHTML))
}

const demoStoreHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>FlowProof Demo Store</title>
<style>
:root{color-scheme:dark;font-family:Inter,ui-sans-serif,system-ui,sans-serif;background:#020617;color:#f8fafc}*{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;background:radial-gradient(circle at top,#172554 0,#020617 48%)}main{width:min(920px,calc(100% - 32px));display:grid;grid-template-columns:1.25fr .75fr;gap:20px}.card{background:#0f172a;border:1px solid #334155;border-radius:20px;padding:24px;box-shadow:0 22px 60px rgba(0,0,0,.35)}.eyebrow{font-size:12px;text-transform:uppercase;letter-spacing:.16em;color:#86efac}.product{display:flex;gap:18px;align-items:center;margin:26px 0}.art{width:116px;height:116px;border-radius:18px;background:linear-gradient(135deg,#14532d,#16a34a);display:grid;place-items:center;font-weight:800;font-size:28px}.muted{color:#94a3b8}button,input{font:inherit}button{cursor:pointer;border:0;border-radius:12px;padding:12px 16px;font-weight:700;background:#16a34a;color:#052e16}button.secondary{background:#1e293b;color:#f8fafc;border:1px solid #475569}input{width:100%;border:1px solid #475569;border-radius:12px;background:#020617;color:#f8fafc;padding:12px;margin:8px 0 10px}button:focus-visible,input:focus-visible{outline:3px solid #86efac;outline-offset:2px}.row{display:flex;align-items:center;justify-content:space-between;gap:12px;margin:12px 0}.total{font-size:28px;font-weight:800}.confirmation{min-height:24px;color:#86efac;font-weight:700}.status{border-top:1px solid #334155;margin-top:16px;padding-top:16px;color:#cbd5e1}@media(max-width:720px){main{grid-template-columns:1fr}}@media(prefers-reduced-motion:no-preference){button{transition:transform .16s ease,filter .16s ease}button:hover{filter:brightness(1.08);transform:translateY(-1px)}}
</style>
</head>
<body>
<main>
<section class="card">
<div class="eyebrow">FlowProof controlled target</div>
<h1>Checkout regression fixture</h1>
<p class="muted">A deterministic storefront used to prove stale-selector failure, evidence capture, and safe recovery.</p>
<div class="product"><div class="art">FP</div><div><h2>Agent QA Toolkit</h2><p class="muted">Browser workflow fixture</p><strong>$100.00</strong></div></div>
<button type="button" data-testid="add-to-cart">Add to cart</button>
<div class="status" id="cart-state" aria-live="polite">Cart is empty</div>
</section>
<aside class="card">
<div class="eyebrow">Order summary</div>
<h2>Checkout</h2>
<label for="coupon">Coupon</label>
<input id="coupon" name="coupon" autocomplete="off" placeholder="Enter coupon">
<button type="button" class="secondary" data-testid="apply-coupon">Apply coupon</button>
<div class="row"><span>Discount</span><strong id="discount">$0.00</strong></div>
<div class="row"><span>Total</span><span class="total" id="total">$100.00</span></div>
<button type="button" data-testid="checkout-submit">Complete checkout</button>
<p class="confirmation" id="confirmation" aria-live="polite"></p>
<p class="muted">Demo coupon: <strong>SAVE20</strong></p>
</aside>
</main>
<script>
(() => {
  let inCart = false;
  let discounted = false;
  const cartState = document.querySelector('#cart-state');
  const coupon = document.querySelector('[name="coupon"]');
  document.querySelector('[data-testid="add-to-cart"]').addEventListener('click', () => {
    inCart = true; cartState.textContent = '1 item in cart · ready for checkout';
  });
  document.querySelector('[data-testid="apply-coupon"]').addEventListener('click', () => {
    discounted = coupon.value.trim().toUpperCase() === 'SAVE20';
    document.querySelector('#discount').textContent = discounted ? '-$20.00' : '$0.00';
    document.querySelector('#total').textContent = discounted ? '$80.00' : '$100.00';
    cartState.textContent = discounted ? 'Coupon SAVE20 applied · total verified at $80.00' : 'Coupon not applied';
  });
  document.querySelector('[data-testid="checkout-submit"]').addEventListener('click', () => {
    document.querySelector('#confirmation').textContent = inCart && discounted ? 'Order confirmed · FP-2048' : 'Checkout prerequisites not satisfied';
  });
})();
</script>
</body>
</html>`
