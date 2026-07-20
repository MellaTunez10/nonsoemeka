import React, { useRef } from 'react';
import { useReactToPrint } from 'react-to-print';
import { Receipt } from '../../types';
import { formatMoney } from '../../lib/money';
import { Printer, X, CheckCircle2 } from 'lucide-react';

interface ReceiptModalProps {
  receipt: Receipt | null;
  onClose: () => void;
}

export const ReceiptModal: React.FC<ReceiptModalProps> = ({ receipt, onClose }) => {
  const contentRef = useRef<HTMLDivElement>(null);

  const reactToPrintFn = useReactToPrint({
    contentRef,
  });

  if (!receipt) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
      <div className="bg-slate-900 border border-slate-700 rounded-2xl w-full max-w-md overflow-hidden shadow-2xl animate-in fade-in zoom-in-95 duration-200">
        <div className="p-4 bg-slate-800/80 border-b border-slate-700 flex items-center justify-between">
          <div className="flex items-center gap-2 text-emerald-400 font-semibold">
            <CheckCircle2 className="w-5 h-5" />
            <span>Transaction Complete</span>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg hover:bg-slate-700 text-slate-400 hover:text-white transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Printable Area */}
        <div className="p-6 max-h-[70vh] overflow-y-auto" ref={contentRef}>
          <div className="text-center pb-4 border-b border-dashed border-slate-700">
            <h2 className="text-xl font-bold text-slate-100 uppercase tracking-wide">
              {receipt.pharmacy_name || 'Nonsoemeka Pharmacy'}
            </h2>
            <p className="text-xs text-slate-400 mt-1">Official Sales Receipt</p>
            <div className="text-xs text-slate-500 mt-2 space-y-0.5">
              <p>Receipt ID: {receipt.id.slice(0, 8)}</p>
              <p>Date: {new Date(receipt.issued_at).toLocaleString()}</p>
              <p>Staff: {receipt.staff_name}</p>
            </div>
          </div>

          <div className="py-4 border-b border-dashed border-slate-700">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="text-slate-400 uppercase border-b border-slate-800 pb-1">
                  <th className="py-1">Item</th>
                  <th className="py-1 text-center">Qty</th>
                  <th className="py-1 text-right">Price</th>
                  <th className="py-1 text-right">Total</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/50 text-slate-200">
                {receipt.items.map((item, idx) => (
                  <tr key={idx}>
                    <td className="py-2 pr-1">
                      <div className="font-medium">{item.product_name}</div>
                      <div className="text-[10px] text-slate-500">Batch: {item.batch_number}</div>
                    </td>
                    <td className="py-2 text-center text-slate-300">{item.quantity}</td>
                    <td className="py-2 text-right text-slate-400">{formatMoney(item.unit_price)}</td>
                    <td className="py-2 text-right font-medium text-slate-200">{formatMoney(item.total_price)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="pt-4 space-y-1">
            <div className="flex justify-between items-center text-base font-bold text-slate-100">
              <span>TOTAL</span>
              <span className="text-emerald-400 text-xl">{formatMoney(receipt.total_amount)}</span>
            </div>
          </div>

          {receipt.footer_text && (
            <div className="mt-6 text-center text-xs text-slate-500 border-t border-slate-800 pt-3 italic">
              {receipt.footer_text}
            </div>
          )}
        </div>

        {/* Action Buttons */}
        <div className="p-4 bg-slate-800/80 border-t border-slate-700 flex gap-3">
          <button
            onClick={() => reactToPrintFn()}
            className="flex-1 py-2.5 px-4 bg-emerald-600 hover:bg-emerald-500 text-white font-medium rounded-xl flex items-center justify-center gap-2 transition-all shadow-lg shadow-emerald-900/30"
          >
            <Printer className="w-4 h-4" />
            <span>Print Receipt</span>
          </button>
          <button
            onClick={onClose}
            className="py-2.5 px-4 bg-slate-700 hover:bg-slate-600 text-slate-200 font-medium rounded-xl transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
