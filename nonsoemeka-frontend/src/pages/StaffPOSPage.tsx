import React, { useState, useEffect, useRef } from 'react';
import { useProducts, useCheckout } from '../hooks/usePharmacy';
import { Product, Receipt } from '../types';
import { Money, formatMoney } from '../lib/money';
import { ReceiptModal } from '../components/receipts/ReceiptModal';
import {
  Search,
  ShoppingCart,
  Trash2,
  Plus,
  Minus,
  CheckCircle,
  AlertCircle,
  Barcode,
  Package,
  Layers,
} from 'lucide-react';

interface CartItem {
  product: Product;
  quantity: number;
}

export const StaffPOSPage: React.FC = () => {
  const [searchTerm, setSearchTerm] = useState('');
  const [cart, setCart] = useState<CartItem[]>([]);
  const [idempotencyKey, setIdempotencyKey] = useState<string>(() => crypto.randomUUID());
  const [receipt, setReceipt] = useState<Receipt | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const searchInputRef = useRef<HTMLInputElement>(null);

  const { data: productsData, isLoading: isProductsLoading } = useProducts(1, searchTerm);
  const checkoutMutation = useCheckout();

  // Focus search input on mount
  useEffect(() => {
    if (searchInputRef.current) {
      searchInputRef.current.focus();
    }
  }, []);

  const addToCart = (product: Product) => {
    if (!product.total_quantity || product.total_quantity <= 0) {
      setErrorMsg(`"${product.name}" is out of stock!`);
      return;
    }

    setErrorMsg(null);
    setCart((prev) => {
      const existing = prev.find((item) => item.product.id === product.id);
      if (existing) {
        if (existing.quantity >= (product.total_quantity || 0)) {
          setErrorMsg(`Cannot add more than available stock (${product.total_quantity}).`);
          return prev;
        }
        return prev.map((item) =>
          item.product.id === product.id ? { ...item, quantity: item.quantity + 1 } : item
        );
      }
      return [...prev, { product, quantity: 1 }];
    });
  };

  const updateQuantity = (productId: string, newQty: number) => {
    setErrorMsg(null);
    if (newQty <= 0) {
      removeFromCart(productId);
      return;
    }

    setCart((prev) =>
      prev.map((item) => {
        if (item.product.id === productId) {
          const maxStock = item.product.total_quantity || 0;
          if (newQty > maxStock) {
            setErrorMsg(`Max stock available is ${maxStock}`);
            return item;
          }
          return { ...item, quantity: newQty };
        }
        return item;
      })
    );
  };

  const removeFromCart = (productId: string) => {
    setCart((prev) => prev.filter((item) => item.product.id !== productId));
  };

  const clearCart = () => {
    setCart([]);
    setIdempotencyKey(crypto.randomUUID());
    setErrorMsg(null);
  };

  // Compute total money using decimal.js Money class
  const cartTotal = cart.reduce((acc, item) => {
    const itemPrice = Money.from(item.product.selling_price || '0');
    return acc.add(itemPrice.mul(item.quantity));
  }, Money.zero());

  const handleCheckout = async () => {
    if (cart.length === 0) return;
    setErrorMsg(null);

    try {
      const items = cart.map((item) => ({
        product_id: item.product.id,
        quantity: item.quantity,
      }));

      const resReceipt = await checkoutMutation.mutateAsync({
        idempotency_key: idempotencyKey,
        items,
      });

      setReceipt(resReceipt);
      // Reset cart and generate new idempotency key on success
      setCart([]);
      setIdempotencyKey(crypto.randomUUID());
    } catch (err: unknown) {
      if (err instanceof Error) {
        setErrorMsg(err.message);
      } else {
        setErrorMsg('Checkout failed.');
      }
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 flex items-center gap-2">
            <ShoppingCart className="w-7 h-7 text-emerald-400" />
            POS Checkout Terminal
          </h1>
          <p className="text-sm text-slate-400">Atomic FEFO stock dispatching with receipt generation</p>
        </div>

        <div className="text-right hidden sm:block">
          <div className="text-xs text-slate-500 font-mono">
            Session Key: <span className="text-slate-400">{idempotencyKey.slice(0, 13)}...</span>
          </div>
        </div>
      </div>

      {errorMsg && (
        <div className="mb-6 p-4 rounded-2xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-sm flex items-center justify-between animate-in fade-in duration-200">
          <div className="flex items-center gap-3">
            <AlertCircle className="w-5 h-5 text-rose-400 shrink-0" />
            <span>{errorMsg}</span>
          </div>
          <button onClick={() => setErrorMsg(null)} className="text-xs text-rose-400 underline">
            Dismiss
          </button>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column: Product Search & Grid */}
        <div className="lg:col-span-7 space-y-4">
          <div className="relative">
            <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
              <Search className="w-5 h-5" />
            </div>
            <input
              ref={searchInputRef}
              type="text"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              placeholder="Search product name or scan SKU barcode..."
              className="w-full pl-11 pr-4 py-3 bg-slate-900/90 border border-slate-700/80 rounded-2xl text-slate-100 placeholder-slate-500 focus:outline-none focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500 transition-all text-sm shadow-inner"
            />
            <div className="absolute inset-y-0 right-0 pr-3.5 flex items-center pointer-events-none text-slate-500 text-xs">
              <Barcode className="w-4 h-4 mr-1" />
              Scanner Ready
            </div>
          </div>

          {/* Product Grid */}
          <div className="bg-slate-900/60 border border-slate-800 rounded-3xl p-4 min-h-[450px] max-h-[600px] overflow-y-auto">
            {isProductsLoading ? (
              <div className="flex items-center justify-center h-64 text-slate-400">
                Loading products...
              </div>
            ) : productsData?.data?.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-64 text-slate-500">
                <Package className="w-12 h-12 mb-2 opacity-50" />
                <p>No active products found matching search.</p>
              </div>
            ) : (
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                {productsData?.data.map((product) => {
                  const outOfStock = !product.total_quantity || product.total_quantity <= 0;
                  return (
                    <button
                      key={product.id}
                      onClick={() => addToCart(product)}
                      disabled={outOfStock}
                      className={`text-left p-4 rounded-2xl border transition-all flex flex-col justify-between group ${
                        outOfStock
                          ? 'bg-slate-950/40 border-slate-900 opacity-50 cursor-not-allowed'
                          : 'bg-slate-800/60 border-slate-700/60 hover:bg-slate-800 hover:border-emerald-500/50 shadow-md'
                      }`}
                    >
                      <div>
                        <div className="flex items-start justify-between">
                          <h3 className="font-semibold text-slate-100 text-sm group-hover:text-emerald-400 transition-colors">
                            {product.name}
                          </h3>
                          <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-slate-900 text-slate-400">
                            {product.sku}
                          </span>
                        </div>
                        {product.description && (
                          <p className="text-xs text-slate-400 mt-1 line-clamp-1">{product.description}</p>
                        )}
                      </div>

                      <div className="mt-4 flex items-center justify-between border-t border-slate-800/80 pt-3">
                        <div className="text-xs">
                          <span className="text-slate-400">Stock: </span>
                          <span
                            className={`font-semibold ${
                              outOfStock
                                ? 'text-rose-400'
                                : (product.total_quantity || 0) < 10
                                ? 'text-amber-400'
                                : 'text-emerald-400'
                            }`}
                          >
                            {product.total_quantity || 0} units
                          </span>
                        </div>
                        <div className="text-base font-bold text-slate-100">
                          {formatMoney(product.selling_price || '0')}
                        </div>
                      </div>
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        </div>

        {/* Right Column: Cart & Summary */}
        <div className="lg:col-span-5">
          <div className="bg-slate-900/90 border border-slate-800 rounded-3xl p-6 flex flex-col justify-between h-full min-h-[550px] shadow-2xl">
            <div>
              <div className="flex items-center justify-between pb-4 border-b border-slate-800">
                <div className="flex items-center gap-2">
                  <Layers className="w-5 h-5 text-emerald-400" />
                  <h2 className="font-bold text-slate-100 text-lg">Current Cart</h2>
                </div>
                {cart.length > 0 && (
                  <button
                    onClick={clearCart}
                    className="text-xs text-rose-400 hover:text-rose-300 flex items-center gap-1"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                    Clear
                  </button>
                )}
              </div>

              {/* Cart List */}
              <div className="divide-y divide-slate-800/80 max-h-[350px] overflow-y-auto my-4 pr-1">
                {cart.length === 0 ? (
                  <div className="text-center py-16 text-slate-500">
                    <ShoppingCart className="w-10 h-10 mx-auto mb-2 opacity-40" />
                    <p className="text-sm">Cart is empty.</p>
                    <p className="text-xs mt-1 text-slate-600">Scan barcode or click items to add.</p>
                  </div>
                ) : (
                  cart.map((item) => {
                    const lineTotal = Money.from(item.product.selling_price || '0').mul(item.quantity);
                    return (
                      <div key={item.product.id} className="py-3 flex items-center justify-between gap-3">
                        <div className="flex-1 min-w-0">
                          <h4 className="text-sm font-medium text-slate-200 truncate">{item.product.name}</h4>
                          <div className="text-xs text-slate-400 font-mono">
                            {formatMoney(item.product.selling_price || '0')} each
                          </div>
                        </div>

                        {/* Quantity Controls */}
                        <div className="flex items-center gap-2 bg-slate-800 rounded-xl p-1 border border-slate-700">
                          <button
                            onClick={() => updateQuantity(item.product.id, item.quantity - 1)}
                            className="p-1 rounded-lg hover:bg-slate-700 text-slate-300 transition-colors"
                          >
                            <Minus className="w-3.5 h-3.5" />
                          </button>
                          <span className="text-xs font-semibold w-6 text-center text-slate-100">
                            {item.quantity}
                          </span>
                          <button
                            onClick={() => updateQuantity(item.product.id, item.quantity + 1)}
                            className="p-1 rounded-lg hover:bg-slate-700 text-slate-300 transition-colors"
                          >
                            <Plus className="w-3.5 h-3.5" />
                          </button>
                        </div>

                        <div className="text-right min-w-[70px]">
                          <div className="text-sm font-bold text-slate-100">{lineTotal.formatCurrency()}</div>
                        </div>
                      </div>
                    );
                  })
                )}
              </div>
            </div>

            {/* Total & Checkout Button */}
            <div className="border-t border-slate-800 pt-4 space-y-4">
              <div className="flex justify-between items-center text-slate-300 text-sm">
                <span>Total Items:</span>
                <span className="font-semibold text-slate-100">
                  {cart.reduce((sum, item) => sum + item.quantity, 0)}
                </span>
              </div>

              <div className="flex justify-between items-center text-lg font-bold text-slate-100 pt-2 border-t border-dashed border-slate-800">
                <span>Grand Total:</span>
                <span className="text-2xl text-emerald-400">{cartTotal.formatCurrency()}</span>
              </div>

              <button
                onClick={handleCheckout}
                disabled={cart.length === 0 || checkoutMutation.isPending}
                className="w-full py-4 bg-gradient-to-r from-emerald-600 to-teal-500 hover:from-emerald-500 hover:to-teal-400 disabled:opacity-50 text-white font-bold rounded-2xl shadow-xl shadow-emerald-950/60 flex items-center justify-center gap-2 text-base transition-all glow-emerald"
              >
                <CheckCircle className="w-5 h-5" />
                <span>{checkoutMutation.isPending ? 'Processing Checkout...' : 'Complete Sale & Print Receipt'}</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Receipt Modal */}
      <ReceiptModal receipt={receipt} onClose={() => setReceipt(null)} />
    </div>
  );
};
