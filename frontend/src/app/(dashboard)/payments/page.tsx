"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { CreditCard, RefreshCw } from "lucide-react";
import { Header } from "@/components/layout/header";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { paymentsApi } from "@/lib/api";
import { formatCurrency, formatDate, PAYMENT_STATUS_MAP, truncateId } from "@/lib/utils";
import { cn } from "@/lib/utils";

const METHOD_LABELS: Record<string, string> = {
  card: "Банковская карта",
  bank_transfer: "Банковский перевод",
  sbp: "СБП",
  yookassa: "ЮKassa",
};

export default function PaymentsPage() {
  const [page] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ["payments", page],
    queryFn: () => paymentsApi.list({ page, limit: 20 }),
    select: (res) => res.data,
  });

  const payments = Array.isArray(data?.data) ? data.data : [];

  return (
    <div className="flex flex-col">
      <Header title="Платежи" description="История платёжных транзакций" />
      <div className="p-6 space-y-4">
        <Card>
          <CardHeader className="py-4 px-6 border-b border-border">
            <div className="grid grid-cols-12 gap-4 text-xs font-medium text-muted-foreground uppercase tracking-wider">
              <div className="col-span-3">ID платежа / Заказ</div>
              <div className="col-span-3">Сумма</div>
              <div className="col-span-2">Метод</div>
              <div className="col-span-2">Статус</div>
              <div className="col-span-2">Дата</div>
            </div>
          </CardHeader>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="flex items-center justify-center h-40">
                <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
              </div>
            ) : payments.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-40 text-muted-foreground">
                <CreditCard className="w-8 h-8 mb-2 opacity-30" />
                <p className="text-sm">Платежей нет</p>
              </div>
            ) : (
              payments.map((payment, i) => {
                const statusConfig = PAYMENT_STATUS_MAP[payment.status] ?? {
                  label: payment.status,
                  color: "bg-gray-500/10 text-gray-500 border-gray-500/20",
                };
                return (
                  <motion.div
                    key={payment.id}
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ delay: i * 0.03 }}
                    className="grid grid-cols-12 gap-4 items-center px-6 py-4 border-b border-border last:border-0 hover:bg-muted/30 transition-colors"
                  >
                    <div className="col-span-3">
                      <div className="flex items-center gap-2">
                        <div className="w-7 h-7 rounded-full bg-green-500/10 flex items-center justify-center flex-shrink-0">
                          <CreditCard className="w-3.5 h-3.5 text-green-500" />
                        </div>
                        <div>
                          <p className="text-xs font-mono font-medium">#{truncateId(payment.id)}</p>
                          <p className="text-xs text-muted-foreground">Заказ: #{truncateId(payment.order_id)}</p>
                        </div>
                      </div>
                    </div>
                    <div className="col-span-3">
                      <p className="text-sm font-semibold">{formatCurrency(payment.amount)}</p>
                    </div>
                    <div className="col-span-2">
                      <p className="text-sm">{METHOD_LABELS[payment.method] ?? payment.method}</p>
                    </div>
                    <div className="col-span-2">
                      <span className={cn(
                        "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold",
                        statusConfig.color
                      )}>
                        {statusConfig.label}
                      </span>
                    </div>
                    <div className="col-span-2">
                      <p className="text-xs text-muted-foreground">{formatDate(payment.created_at)}</p>
                    </div>
                  </motion.div>
                );
              })
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
