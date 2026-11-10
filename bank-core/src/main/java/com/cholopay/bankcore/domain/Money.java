package com.cholopay.bankcore.domain;

import java.util.Objects;

public record Money(long minor, String currency) {
    public Money {
        if (minor < 0) throw new IllegalArgumentException("amount_minor no puede ser negativo");
        if (!Objects.equals("USD", currency)) throw new IllegalArgumentException("currency debe ser USD");
    }
    public static Money usd(long minor) { return new Money(minor, "USD"); }
}
