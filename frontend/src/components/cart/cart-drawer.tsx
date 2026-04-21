"use client";

import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { ShoppingBag, X, Trash2, Plus, Minus, CheckCircle2 } from "lucide-react";
import { useMutation } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useCartStore } from "@/store/useStore";
import { ordersApi } from "@/lib/api";
import { formatCurrency, getImageUrl } from "@/lib/utils";

interface CartDrawerProps {
  open: boolean;
  onClose: () => void;
}

export function CartDrawer({ open, onClose }: CartDrawerProps) {
  const { items, removeItem, updateQuantity, clearCart } = useCartStore();
  const [address, setAddress] = useState("");
  const [notes, setNotes] = useState("");
  const [orderDone, setOrderDone] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");

  const total = items.reduce((sum, item) => sum + item.product.price * item.quantity, 0);

  const createOrderMutation = useMutation({
    mutationFn: () =>
      ordersApi.create({
        shipping_address: address,
        notes: notes || undefined,
        items: items.map((item) => ({
          product_id: item.product.id,
          quantity: item.quantity,
          price: item.product.price,
          name: item.product.name,
        })),
      }),
    onSuccess: () => {
      clearCart();
      setOrderDone(true);
      setErrorMsg("");
    },
    onError: (err: unknown) => {
      const axiosErr = err as { response?: { data?: { error?: string } } };
      setErrorMsg(axiosErr?.response?.data?.error || "Ошибка при создании заказа");
    },
  });

  const handleClose = () => {
    setOrderDone(false);
    setErrorMsg("");
    onClose();
  };

  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 bg-black/50"
            onClick={handleClose}
          />
          <motion.div
            initial={{ x: "100%" }}
            animate={{ x: 0 }}
            exit={{ x: "100%" }}
            transition={{ type: "spring", damping: 28, stiffness: 220 }}
            className="fixed right-0 top-0 z-50 h-screen w-full max-w-md bg-card border-l border-border shadow-2xl flex flex-col"
          >
            <div className="flex items-center justify-between p-5 border-b border-border">
              <div className="flex items-center gap-2">
                <ShoppingBag className="w-5 h-5" />
                <h2 className="text-lg font-semibold">Корзина</h2>
                {items.length > 0 && (
                  <span className="text-xs bg-primary text-primary-foreground rounded-full px-2 py-0.5 font-medium">
                    {items.reduce((s, i) => s + i.quantity, 0)}
                  </span>
                )}
              </div>
              <Button variant="ghost" size="icon" onClick={handleClose}>
                <X className="w-4 h-4" />
              </Button>
            </div>

            {orderDone ? (
              <div className="flex-1 flex flex-col items-center justify-center p-8 text-center gap-5">
                <div className="w-16 h-16 rounded-full bg-green-500/10 flex items-center justify-center">
                  <CheckCircle2 className="w-9 h-9 text-green-500" />
                </div>
                <div>
                  <h3 className="text-lg font-semibold mb-1">Заказ оформлен!</h3>
                  <p className="text-sm text-muted-foreground">
                    Статус заказа можно отслеживать в разделе «Заказы»
                  </p>
                </div>
                <Button onClick={handleClose} className="w-full max-w-xs">
                  Отлично
                </Button>
              </div>
            ) : items.length === 0 ? (
              <div className="flex-1 flex flex-col items-center justify-center p-6 text-muted-foreground gap-3">
                <ShoppingBag className="w-12 h-12 opacity-20" />
                <p className="text-sm">Корзина пуста</p>
                <p className="text-xs">Добавьте товары из каталога</p>
              </div>
            ) : (
              <>
                <div className="flex-1 overflow-y-auto p-4 space-y-3">
                  {items.map((item) => (
                    <div
                      key={item.product.id}
                      className="flex gap-3 p-3 bg-muted/30 rounded-lg border border-border/50"
                    >
                      <div className="w-14 h-14 rounded-md bg-muted flex items-center justify-center flex-shrink-0 overflow-hidden">
                        {getImageUrl(item.product.images?.[0]) ? (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img
                            src={getImageUrl(item.product.images[0])!}
                            alt={item.product.name}
                            className="w-full h-full object-cover"
                            onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
                          />
                        ) : (
                          <ShoppingBag className="w-5 h-5 text-muted-foreground/40" />
                        )}
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium truncate">{item.product.name}</p>
                        <p className="text-xs text-muted-foreground">
                          {formatCurrency(item.product.price)} / шт.
                        </p>
                        <div className="flex items-center gap-2 mt-2">
                          <Button
                            size="icon"
                            variant="outline"
                            className="h-6 w-6"
                            onClick={() => updateQuantity(item.product.id, item.quantity - 1)}
                          >
                            <Minus className="w-3 h-3" />
                          </Button>
                          <span className="text-sm w-5 text-center font-medium">{item.quantity}</span>
                          <Button
                            size="icon"
                            variant="outline"
                            className="h-6 w-6"
                            onClick={() => updateQuantity(item.product.id, item.quantity + 1)}
                          >
                            <Plus className="w-3 h-3" />
                          </Button>
                        </div>
                      </div>
                      <div className="flex flex-col items-end justify-between">
                        <Button
                          size="icon"
                          variant="ghost"
                          className="h-6 w-6 text-muted-foreground hover:text-destructive"
                          onClick={() => removeItem(item.product.id)}
                        >
                          <Trash2 className="w-3 h-3" />
                        </Button>
                        <p className="text-sm font-semibold">
                          {formatCurrency(item.product.price * item.quantity)}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>

                <div className="p-5 border-t border-border space-y-4">
                  <div className="flex justify-between text-base font-semibold">
                    <span>Итого</span>
                    <span>{formatCurrency(total)}</span>
                  </div>
                  <div className="space-y-3">
                    <div>
                      <Label className="text-xs mb-1.5 block">Адрес доставки *</Label>
                      <Input
                        placeholder="г. Москва, ул. Примерная, д. 1, кв. 1"
                        value={address}
                        onChange={(e) => setAddress(e.target.value)}
                      />
                    </div>
                    <div>
                      <Label className="text-xs mb-1.5 block">Комментарий (необязательно)</Label>
                      <Input
                        placeholder="Позвонить за 30 минут до доставки..."
                        value={notes}
                        onChange={(e) => setNotes(e.target.value)}
                      />
                    </div>
                  </div>
                  {errorMsg && (
                    <p className="text-xs text-destructive bg-destructive/10 rounded-md px-3 py-2">
                      {errorMsg}
                    </p>
                  )}
                  <Button
                    className="w-full"
                    disabled={!address.trim() || createOrderMutation.isPending}
                    onClick={() => createOrderMutation.mutate()}
                  >
                    {createOrderMutation.isPending ? "Оформление..." : "Оформить заказ"}
                  </Button>
                  <Button
                    variant="ghost"
                    className="w-full text-xs text-muted-foreground h-8"
                    onClick={clearCart}
                  >
                    Очистить корзину
                  </Button>
                </div>
              </>
            )}
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}
