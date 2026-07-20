import Decimal from 'decimal.js';

/**
 * Decimal money utility wrapping decimal.js for financial accuracy.
 * Never use JavaScript standard numbers for arithmetic calculations on money!
 */
export class Money {
  private val: Decimal;

  constructor(amount: string | number | Decimal) {
    this.val = new Decimal(amount || '0');
  }

  static from(amount: string | number | Decimal): Money {
    return new Money(amount);
  }

  static zero(): Money {
    return new Money('0');
  }

  add(other: string | number | Money): Money {
    const o = other instanceof Money ? other.val : new Decimal(other);
    return new Money(this.val.plus(o));
  }

  sub(other: string | number | Money): Money {
    const o = other instanceof Money ? other.val : new Decimal(other);
    return new Money(this.val.minus(o));
  }

  mul(qty: number | string): Money {
    return new Money(this.val.times(qty));
  }

  div(divider: number | string): Money {
    return new Money(this.val.dividedBy(divider));
  }

  toStringFixed(decimals = 2): string {
    return this.val.toFixed(decimals);
  }

  formatCurrency(currencySymbol = '₦'): string {
    const parts = this.val.toFixed(2).split('.');
    const integerPart = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ',');
    return `${currencySymbol}${integerPart}.${parts[1]}`;
  }

  isZero(): boolean {
    return this.val.isZero();
  }

  isGreaterThanZero(): boolean {
    return this.val.greaterThan(0);
  }
}

export function formatMoney(amount: string | number, symbol = '₦'): string {
  if (!amount) return `${symbol}0.00`;
  return Money.from(amount).formatCurrency(symbol);
}
