import React, { useState } from 'react';
import {
  useAdminProducts,
  useBatches,
  useCreateProduct,
  useRegisterBatch,
  useAdjustStock,
  useWriteOffStock,
  useSettings,
} from '../hooks/usePharmacy';
import { formatMoney } from '../lib/money';
import {
  PackagePlus,
  PlusCircle,
  Package,
  Layers,
  Sliders,
  AlertOctagon,
  X,
  Search,
} from 'lucide-react';

export const AdminInventoryPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'products' | 'batches'>('products');
  const [searchTerm, setSearchTerm] = useState('');
  const [showAddProduct, setShowAddProduct] = useState(false);
  const [showAddBatch, setShowAddBatch] = useState(false);
  const [selectedBatchId, setSelectedBatchId] = useState<string | null>(null);
  const [showAdjustModal, setShowAdjustModal] = useState(false);
  const [showWriteOffModal, setShowWriteOffModal] = useState(false);

  // Forms state
  const [productForm, setProductForm] = useState({ name: '', sku: '', description: '' });
  const [batchForm, setBatchForm] = useState({
    product_id: '',
    batch_number: '',
    quantity_received: 100,
    expiry_date: '',
    cost_price: '100.00',
    markup_percentage: '25.00',
  });
  const [adjustForm, setAdjustForm] = useState({ quantity_delta: 0, reason: '' });
  const [writeOffForm, setWriteOffForm] = useState({ reason: '' });
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const { data: productsData } = useAdminProducts(1, searchTerm);
  const { data: batchesData } = useBatches(1, searchTerm);
  const { data: settingsData } = useSettings();

  const createProduct = useCreateProduct();
  const registerBatch = useRegisterBatch();
  const adjustStock = useAdjustStock();
  const writeOffStock = useWriteOffStock();

  const handleCreateProduct = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);
    try {
      await createProduct.mutateAsync(productForm);
      setShowAddProduct(false);
      setProductForm({ name: '', sku: '', description: '' });
    } catch (err: unknown) {
      if (err instanceof Error) setErrorMsg(err.message);
    }
  };

  const handleRegisterBatch = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);
    try {
      await registerBatch.mutateAsync({
        ...batchForm,
        markup_percentage: batchForm.markup_percentage || settingsData?.default_markup_percentage,
      });
      setShowAddBatch(false);
      setBatchForm({
        product_id: '',
        batch_number: '',
        quantity_received: 100,
        expiry_date: '',
        cost_price: '100.00',
        markup_percentage: settingsData?.default_markup_percentage || '25.00',
      });
    } catch (err: unknown) {
      if (err instanceof Error) setErrorMsg(err.message);
    }
  };

  const handleAdjustStock = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedBatchId) return;
    setErrorMsg(null);
    try {
      await adjustStock.mutateAsync({ id: selectedBatchId, req: adjustForm });
      setShowAdjustModal(false);
      setAdjustForm({ quantity_delta: 0, reason: '' });
    } catch (err: unknown) {
      if (err instanceof Error) setErrorMsg(err.message);
    }
  };

  const handleWriteOffStock = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedBatchId) return;
    setErrorMsg(null);
    try {
      await writeOffStock.mutateAsync({ id: selectedBatchId, req: writeOffForm });
      setShowWriteOffModal(false);
      setWriteOffForm({ reason: '' });
    } catch (err: unknown) {
      if (err instanceof Error) setErrorMsg(err.message);
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">
      {/* Top Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 flex items-center gap-2">
            <Package className="w-7 h-7 text-emerald-400" />
            Inventory & Batch Portal
          </h1>
          <p className="text-sm text-slate-400">Manage pharmaceutical products, batch intakes, and stock adjustments</p>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={() => setShowAddProduct(true)}
            className="py-2.5 px-4 bg-slate-800 hover:bg-slate-700 border border-slate-700 text-slate-200 text-sm font-medium rounded-xl flex items-center gap-2 transition-colors"
          >
            <PackagePlus className="w-4 h-4 text-emerald-400" />
            <span>New Product</span>
          </button>
          <button
            onClick={() => setShowAddBatch(true)}
            className="py-2.5 px-4 bg-emerald-600 hover:bg-emerald-500 text-white text-sm font-semibold rounded-xl flex items-center gap-2 transition-all shadow-lg shadow-emerald-950/50"
          >
            <PlusCircle className="w-4 h-4" />
            <span>Register Batch</span>
          </button>
        </div>
      </div>

      {errorMsg && (
        <div className="p-4 rounded-2xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-sm">
          {errorMsg}
        </div>
      )}

      {/* Tabs & Search */}
      <div className="flex flex-col sm:flex-row items-center justify-between gap-4 border-b border-slate-800 pb-4">
        <div className="flex items-center gap-2 bg-slate-900/80 p-1 rounded-xl border border-slate-800 w-full sm:w-auto">
          <button
            onClick={() => setActiveTab('products')}
            className={`flex-1 sm:flex-none px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              activeTab === 'products' ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30' : 'text-slate-400'
            }`}
          >
            Products ({productsData?.pagination?.total_items || 0})
          </button>
          <button
            onClick={() => setActiveTab('batches')}
            className={`flex-1 sm:flex-none px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              activeTab === 'batches' ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30' : 'text-slate-400'
            }`}
          >
            Batches ({batchesData?.pagination?.total_items || 0})
          </button>
        </div>

        <div className="relative w-full sm:w-72">
          <Search className="w-4 h-4 absolute left-3 top-3 text-slate-400" />
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="Search items..."
            className="w-full pl-9 pr-4 py-2 bg-slate-900 border border-slate-800 rounded-xl text-slate-200 text-sm focus:outline-none focus:border-emerald-500"
          />
        </div>
      </div>

      {/* Content Tables */}
      {activeTab === 'products' ? (
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl overflow-hidden shadow-xl">
          <table className="w-full text-left text-sm text-slate-300">
            <thead className="bg-slate-950 text-slate-400 uppercase text-xs">
              <tr>
                <th className="py-3.5 px-4">Product Name</th>
                <th className="py-3.5 px-4">SKU</th>
                <th className="py-3.5 px-4">Total Stock</th>
                <th className="py-3.5 px-4">Selling Price</th>
                <th className="py-3.5 px-4">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {productsData?.data.map((p) => (
                <tr key={p.id} className="hover:bg-slate-800/40">
                  <td className="py-3 px-4 font-semibold text-slate-100">{p.name}</td>
                  <td className="py-3 px-4 font-mono text-slate-400 text-xs">{p.sku}</td>
                  <td className="py-3 px-4 font-medium text-emerald-400">{p.total_quantity || 0} units</td>
                  <td className="py-3 px-4 font-bold text-slate-200">{formatMoney(p.selling_price || '0')}</td>
                  <td className="py-3 px-4">
                    <span className="px-2 py-0.5 rounded text-[10px] uppercase font-bold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                      Active
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl overflow-hidden shadow-xl">
          <table className="w-full text-left text-sm text-slate-300">
            <thead className="bg-slate-950 text-slate-400 uppercase text-xs">
              <tr>
                <th className="py-3.5 px-4">Batch #</th>
                <th className="py-3.5 px-4">Product Name</th>
                <th className="py-3.5 px-4">Remaining</th>
                <th className="py-3.5 px-4">Expiry Date</th>
                <th className="py-3.5 px-4">Cost Price</th>
                <th className="py-3.5 px-4">Selling Price</th>
                <th className="py-3.5 px-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {batchesData?.data.map((b) => (
                <tr key={b.id} className="hover:bg-slate-800/40">
                  <td className="py-3 px-4 font-mono text-slate-100 font-semibold">{b.batch_number}</td>
                  <td className="py-3 px-4 text-slate-200">{b.product_name || 'N/A'}</td>
                  <td className="py-3 px-4 font-medium text-emerald-400">{b.quantity_remaining} units</td>
                  <td className="py-3 px-4 text-slate-300 font-mono text-xs">{b.expiry_date}</td>
                  <td className="py-3 px-4 text-slate-400">{formatMoney(b.cost_price)}</td>
                  <td className="py-3 px-4 font-bold text-slate-100">{formatMoney(b.selling_price)}</td>
                  <td className="py-3 px-4 text-right space-x-2">
                    <button
                      onClick={() => {
                        setSelectedBatchId(b.id);
                        setShowAdjustModal(true);
                      }}
                      className="p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300"
                      title="Adjust Stock"
                    >
                      <Sliders className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => {
                        setSelectedBatchId(b.id);
                        setShowWriteOffModal(true);
                      }}
                      className="p-1.5 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/20"
                      title="Write Off Stock"
                    >
                      <AlertOctagon className="w-4 h-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* New Product Modal */}
      {showAddProduct && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="bg-slate-900 border border-slate-700 rounded-2xl w-full max-w-md p-6">
            <div className="flex justify-between items-center mb-4">
              <h3 className="font-bold text-lg text-slate-100">Create New Product</h3>
              <button onClick={() => setShowAddProduct(false)} className="text-slate-400 hover:text-white">
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleCreateProduct} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-300 uppercase mb-1">Product Name</label>
                <input
                  type="text"
                  required
                  value={productForm.name}
                  onChange={(e) => setProductForm({ ...productForm, name: e.target.value })}
                  className="w-full p-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
                  placeholder="e.g. Paracetamol 500mg"
                />
              </div>
              <div>
                <label className="block text-xs font-semibold text-slate-300 uppercase mb-1">SKU Barcode</label>
                <input
                  type="text"
                  required
                  value={productForm.sku}
                  onChange={(e) => setProductForm({ ...productForm, sku: e.target.value })}
                  className="w-full p-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm font-mono"
                  placeholder="e.g. PARA-500MG"
                />
              </div>
              <button
                type="submit"
                className="w-full py-3 bg-emerald-600 hover:bg-emerald-500 text-white font-bold rounded-xl"
              >
                Create Product
              </button>
            </form>
          </div>
        </div>
      )}

      {/* New Batch Modal */}
      {showAddBatch && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="bg-slate-900 border border-slate-700 rounded-2xl w-full max-w-lg p-6">
            <div className="flex justify-between items-center mb-4">
              <h3 className="font-bold text-lg text-slate-100">Register Product Batch</h3>
              <button onClick={() => setShowAddBatch(false)} className="text-slate-400 hover:text-white">
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleRegisterBatch} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-300 uppercase mb-1">Select Product</label>
                <select
                  required
                  value={batchForm.product_id}
                  onChange={(e) => setBatchForm({ ...batchForm, product_id: e.target.value })}
                  className="w-full p-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
                >
                  <option value="">-- Choose Product --</option>
                  {productsData?.data.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name} ({p.sku})
                    </option>
                  ))}
                </select>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-300 uppercase mb-1">Batch Number</label>
                  <input
                    type="text"
                    required
                    value={batchForm.batch_number}
                    onChange={(e) => setBatchForm({ ...batchForm, batch_number: e.target.value })}
                    className="w-full p-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm font-mono"
                    placeholder="BATCH-2026-001"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-300 uppercase mb-1">Quantity Received</label>
                  <input
                    type="number"
                    required
                    min={1}
                    value={batchForm.quantity_received}
                    onChange={(e) => setBatchForm({ ...batchForm, quantity_received: parseInt(e.target.value) })}
                    className="w-full p-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-300 uppercase mb-1">Expiry Date</label>
                  <input
                    type="date"
                    required
                    value={batchForm.expiry_date}
                    onChange={(e) => setBatchForm({ ...batchForm, expiry_date: e.target.value })}
                    className="w-full p-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-300 uppercase mb-1">Cost Price (₦)</label>
                  <input
                    type="text"
                    required
                    value={batchForm.cost_price}
                    onChange={(e) => setBatchForm({ ...batchForm, cost_price: e.target.value })}
                    className="w-full p-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-300 uppercase mb-1">Markup %</label>
                <input
                  type="text"
                  value={batchForm.markup_percentage}
                  onChange={(e) => setBatchForm({ ...batchForm, markup_percentage: e.target.value })}
                  className="w-full p-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
                  placeholder="25.00"
                />
                <p className="text-[10px] text-slate-500 mt-1 italic">
                  Note: Selling price is generated by PostgreSQL using cost price and markup.
                </p>
              </div>

              <button
                type="submit"
                className="w-full py-3 bg-emerald-600 hover:bg-emerald-500 text-white font-bold rounded-xl"
              >
                Register Batch
              </button>
            </form>
          </div>
        </div>
      )}

      {/* Adjust Modal */}
      {showAdjustModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="bg-slate-900 border border-slate-700 rounded-2xl w-full max-w-md p-6">
            <h3 className="font-bold text-lg text-slate-100 mb-4">Adjust Batch Quantity</h3>
            <form onSubmit={handleAdjustStock} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-300 uppercase mb-1">Quantity Delta (+/-)</label>
                <input
                  type="number"
                  required
                  value={adjustForm.quantity_delta}
                  onChange={(e) => setAdjustForm({ ...adjustForm, quantity_delta: parseInt(e.target.value) })}
                  className="w-full p-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
                  placeholder="e.g. -5 or +10"
                />
              </div>
              <div>
                <label className="block text-xs font-semibold text-slate-300 uppercase mb-1">Reason (Mandatory)</label>
                <textarea
                  required
                  rows={3}
                  value={adjustForm.reason}
                  onChange={(e) => setAdjustForm({ ...adjustForm, reason: e.target.value })}
                  className="w-full p-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
                  placeholder="Inventory audit discrepancy..."
                />
              </div>
              <div className="flex gap-2">
                <button type="submit" className="flex-1 py-2.5 bg-emerald-600 text-white font-bold rounded-xl">
                  Confirm
                </button>
                <button
                  type="button"
                  onClick={() => setShowAdjustModal(false)}
                  className="py-2.5 px-4 bg-slate-800 text-slate-300 rounded-xl"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Write Off Modal */}
      {showWriteOffModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="bg-slate-900 border border-slate-700 rounded-2xl w-full max-w-md p-6">
            <h3 className="font-bold text-lg text-rose-400 mb-4">Write Off Batch Stock</h3>
            <form onSubmit={handleWriteOffStock} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-300 uppercase mb-1">Reason for Write-Off</label>
                <textarea
                  required
                  rows={3}
                  value={writeOffForm.reason}
                  onChange={(e) => setWriteOffForm({ ...writeOffForm, reason: e.target.value })}
                  className="w-full p-2.5 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
                  placeholder="Expired or damaged batch..."
                />
              </div>
              <div className="flex gap-2">
                <button type="submit" className="flex-1 py-2.5 bg-rose-600 text-white font-bold rounded-xl">
                  Write Off Stock
                </button>
                <button
                  type="button"
                  onClick={() => setShowWriteOffModal(false)}
                  className="py-2.5 px-4 bg-slate-800 text-slate-300 rounded-xl"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
