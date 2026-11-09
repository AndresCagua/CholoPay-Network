package com.cholopay.bankcore.domain;

import java.util.Objects;

public record Money(long minor, String currency) {
    public Money {
        if (minor < 0) throw new IllegalArgumentException("amount_minor must not be negative");
        if (!Objects.equals("USD", currency)) throw new IllegalArgumentException("currency must be USD");
    }
    public static Money usd(long minor) { return new Money(minor, "USD"); }
}
