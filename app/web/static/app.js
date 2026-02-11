/* Micro Trading - Client JS */

// Toggle limit price field based on order type
document.addEventListener('DOMContentLoaded', () => {
    const orderType = document.getElementById('order-type');
    const limitPrice = document.getElementById('limit-price');

    if (orderType && limitPrice) {
        const toggle = () => {
            limitPrice.disabled = orderType.value !== 'LIMIT';
            if (limitPrice.disabled) limitPrice.value = '';
        };
        orderType.addEventListener('change', toggle);
        toggle();
    }
});
