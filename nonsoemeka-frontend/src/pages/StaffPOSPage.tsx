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
  const [activeTab, setActiveTab] = useState<'products' | 'cart'>('products');

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

  const totalCartCount = cart.reduce((sum, item) => sum + item.quantity, 0);

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
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 sm:py-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-4 sm:mb-6 gap-2">
        <div>
          <h1 className="text-xl sm:text-2xl font-bold dark:text-slate-100 light:text-slate-900 flex items-center gap-2">
            <ShoppingCart className="w-6 h-6 sm:w-7 sm:h-7 text-emerald-500" />
            POS Checkout Terminal
          </h1>
          <p className="text-xs sm:text-sm dark:text-slate-400 light:text-slate-600">Atomic FEFO stock dispatching with receipt generation</p>
        </div>

        <div className="text-left sm:text-right">
          <div className="text-[11px] sm:text-xs dark:text-slate-500 light:text-slate-500 font-mono">
            Session Key: <span className="dark:text-slate-400 light:text-slate-700">{idempotencyKey.slice(0, 13)}...</span>
          </div>
        </div>
      </div>

      {/* Mobile Tab Switcher */}
      <div className="flex lg:hidden items-center gap-2 mb-4 p-1 dark:bg-slate-900 light:bg-slate-200 rounded-2xl border dark:border-slate-800 light:border-slate-300">
        <button
          onClick={() => setActiveTab('products')}
          className={`flex-1 py-2.5 px-3 rounded-xl text-sm font-semibold transition-all flex items-center justify-center gap-2 ${
            activeTab === 'products'
              ? 'bg-emerald-600 text-white shadow-md'
              : 'dark:text-slate-400 light:text-slate-700 hover:text-slate-900'
          }`}
        >
          <Package className="w-4 h-4" />
          <span>Products</span>
        </button>
        <button
          onClick={() => setActiveTab('cart')}
          className={`flex-1 py-2.5 px-3 rounded-xl text-sm font-semibold transition-all flex items-center justify-center gap-2 relative ${
            activeTab === 'cart'
              ? 'bg-emerald-600 text-white shadow-md'
              : 'dark:text-slate-400 light:text-slate-700 hover:text-slate-900'
          }`}
        >
          <Layers className="w-4 h-4" />
          <span>Cart ({totalCartCount})</span>
          {totalCartCount > 0 && activeTab !== 'cart' && (
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-ping absolute top-2 right-4" />
          )}
        </button>
      </div>

      {errorMsg && (
        <div className="mb-4 sm:mb-6 p-4 rounded-2xl bg-rose-500/10 border border-rose-500/30 text-rose-500 text-sm flex items-center justify-between animate-in fade-in duration-200">
          <div className="flex items-center gap-3">
            <AlertCircle className="w-5 h-5 text-rose-500 shrink-0" />
            <span>{errorMsg}</span>
          </div>
          <button onClick={() => setErrorMsg(null)} className="text-xs text-rose-500 underline">
            Dismiss
          </button>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column: Product Search & Grid */}
        <div className={`lg:col-span-7 space-y-4 ${activeTab === 'cart' ? 'hidden lg:block' : 'block'}`}>
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
              className="w-full pl-11 pr-28 sm:pr-32 py-3 dark:bg-slate-900/90 light:bg-white border dark:border-slate-700/80 light:border-slate-300 rounded-2xl dark:text-slate-100 light:text-slate-900 dark:placeholder-slate-500 light:placeholder-slate-400 focus:outline-none focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500 transition-all text-sm shadow-inner"
            />
            <div className="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none text-slate-400 text-[11px] sm:text-xs">
              <Barcode className="w-4 h-4 mr-1 text-emerald-500" />
              <span className="hidden xs:inline">Scanner Ready</span>
            </div>
          </div>

          {/* Product Grid */}
          <div className="dark:bg-slate-900/60 light:bg-white border dark:border-slate-800 light:border-slate-200 rounded-3xl p-4 min-h-[400px] sm:min-h-[450px] max-h-[600px] overflow-y-auto shadow-sm">
            {isProductsLoading ? (
              <div className="flex items-center justify-center h-64 dark:text-slate-400 light:text-slate-500">
                Loading products...
              </div>
            ) : productsData?.data?.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-64 dark:text-slate-500 light:text-slate-400">
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
                          ? 'dark:bg-slate-950/40 light:bg-slate-100 dark:border-slate-900 light:border-slate-200 opacity-50 cursor-not-allowed'
                          : 'dark:bg-slate-800/60 light:bg-slate-50 border-slate-700/60 light:border-slate-200 hover:border-emerald-500/50 hover:bg-emerald-500/5 shadow-sm'
                      }`}
                    >
                      <div>
                        <div className="flex items-start justify-between gap-2">
                          <h3 className="font-semibold dark:text-slate-100 light:text-slate-900 text-sm group-hover:text-emerald-500 transition-colors">
                            {product.name}
                          </h3>
                          <span className="text-[10px] font-mono px-2 py-0.5 rounded dark:bg-slate-900 light:bg-slate-200 dark:text-slate-400 light:text-slate-700 shrink-0">
                            {product.sku}
                          </span>
                        </div>
                        {product.description && (
                          <p className="text-xs dark:text-slate-400 light:text-slate-500 mt-1 line-clamp-1">{product.description}</p>
                        )}
                      </div>

                      <div className="mt-4 flex items-center justify-between border-t dark:border-slate-800/80 light:border-slate-200 pt-3">
                        <div className="text-xs">
                          <span className="dark:text-slate-400 light:text-slate-500">Stock: </span>
                          <span
                            className={`font-semibold ${
                              outOfStock
                                ? 'text-rose-500'
                                : (product.total_quantity || 0) < 10
                                ? 'text-amber-500'
                                : 'text-emerald-500'
                            }`}
                          >
                            {product.total_quantity || 0} units
                          </span>
                        </div>
                        <div className="text-base font-bold dark:text-slate-100 light:text-slate-900">
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
        <div className={`lg:col-span-5 ${activeTab === 'products' ? 'hidden lg:block' : 'block'}`}>
          <div className="dark:bg-slate-900/90 light:bg-white border dark:border-slate-800 light:border-slate-200 rounded-3xl p-5 sm:p-6 flex flex-col justify-between h-full min-h-[450px] sm:min-h-[550px] shadow-xl">
            <div>
              <div className="flex items-center justify-between pb-4 border-b dark:border-slate-800 light:border-slate-200">
                <div className="flex items-center gap-2">
                  <Layers className="w-5 h-5 text-emerald-500" />
                  <h2 className="font-bold dark:text-slate-100 light:text-slate-900 text-lg">Current Cart</h2>
                </div>
                {cart.length > 0 && (
                  <button
                    onClick={clearCart}
                    className="text-xs text-rose-500 hover:text-rose-600 flex items-center gap-1 font-medium"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                    Clear
                  </button>
                )}
              </div>

              {/* Cart List */}
              <div className="divide-y dark:divide-slate-800/80 light:divide-slate-100 max-h-[320px] sm:max-h-[350px] overflow-y-auto my-4 pr-1">
                {cart.length === 0 ? (
                  <div className="text-center py-12 sm:py-16 dark:text-slate-500 light:text-slate-400">
                    <ShoppingCart className="w-10 h-10 mx-auto mb-2 opacity-40" />
                    <p className="text-sm">Cart is empty.</p>
                    <p className="text-xs mt-1 opacity-75">Scan barcode or click items to add.</p>
                  </div>
                ) : (
                  cart.map((item) => {
                    const lineTotal = Money.from(item.product.selling_price || '0').mul(item.quantity);
                    return (
                      <div key={item.product.id} className="py-3 flex items-center justify-between gap-3">
                        <div className="flex-1 min-w-0">
                          <h4 className="text-sm font-medium dark:text-slate-200 light:text-slate-800 truncate">{item.product.name}</h4>
                          <div className="text-xs dark:text-slate-400 light:text-slate-500 font-mono">
                            {formatMoney(item.product.selling_price || '0')} each
                          </div>
                        </div>

                        {/* Quantity Controls */}
                        <div className="flex items-center gap-2 dark:bg-slate-800 light:bg-slate-100 rounded-xl p-1 border dark:border-slate-700 light:border-slate-300">
                          <button
                            onClick={() => updateQuantity(item.product.id, item.quantity - 1)}
                            className="p-1 rounded-lg hover:bg-slate-700/50 light:hover:bg-slate-200 dark:text-slate-300 light:text-slate-700 transition-colors"
                          >
                            <Minus className="w-3.5 h-3.5" />
                          </button>
                          <span className="text-xs font-semibold w-6 text-center dark:text-slate-100 light:text-slate-900">
                            {item.quantity}
                          </span>
                          <button
                            onClick={() => updateQuantity(item.product.id, item.quantity + 1)}
                            className="p-1 rounded-lg hover:bg-slate-700/50 light:hover:bg-slate-200 dark:text-slate-300 light:text-slate-700 transition-colors"
                          >
                            <Plus className="w-3.5 h-3.5" />
                          </button>
                        </div>

                        <div className="text-right min-w-[70px]">
                          <div className="text-sm font-bold dark:text-slate-100 light:text-slate-900">{lineTotal.formatCurrency()}</div>
                        </div>
                      </div>
                    );
                  })
                )}
              </div>
            </div>

            {/* Total & Checkout Button */}
            <div className="border-t dark:border-slate-800 light:border-slate-200 pt-4 space-y-4">
              <div className="flex justify-between items-center text-sm dark:text-slate-300 light:text-slate-600">
                <span>Total Items:</span>
                <span className="font-semibold dark:text-slate-100 light:text-slate-900">
                  {totalCartCount}
                </span>
              </div>

              <div className="flex justify-between items-center text-lg font-bold dark:text-slate-100 light:text-slate-900 pt-2 border-t border-dashed dark:border-slate-800 light:border-slate-200">
                <span>Grand Total:</span>
                <span className="text-2xl text-emerald-500 font-extrabold">{cartTotal.formatCurrency()}</span>
              </div>

              <button
                onClick={handleCheckout}
                disabled={cart.length === 0 || checkoutMutation.isPending}
                className="w-full py-4 bg-gradient-to-r from-emerald-600 to-teal-500 hover:from-emerald-500 hover:to-teal-400 disabled:opacity-50 text-white font-bold rounded-2xl shadow-xl flex items-center justify-center gap-2 text-base transition-all glow-emerald"
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
