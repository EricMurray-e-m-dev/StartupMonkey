// BarberBook - Admin Panel
// AI-generated admin panel for today's bookings

const API_BASE_URL = '/api';

document.addEventListener('DOMContentLoaded', () => {
    loadTodayBookings();
});

// Load today's bookings
async function loadTodayBookings() {
    const loading = document.getElementById('loading');
    const container = document.getElementById('bookings-container');
    const noBookings = document.getElementById('no-bookings');
    const tbody = document.getElementById('bookings-tbody');
    
    try {
        // INTENTIONAL FLAW: This endpoint does full table scan on booking_date
        const response = await fetch(`${API_BASE_URL}/bookings/today`);
        
        if (!response.ok) {
            throw new Error('Failed to load bookings');
        }
        
        const bookings = await response.json();
        
        loading.classList.add('d-none');
        
        if (bookings.length === 0) {
            noBookings.classList.remove('d-none');
            return;
        }
        
        // Render bookings table
        tbody.innerHTML = bookings.map(booking => `
            <tr>
                <td>${booking.booking_time}</td>
                <td>${booking.customer_name}</td>
                <td>${booking.customer_phone}</td>
                <td>${booking.barber_name}</td>
                <td>${booking.service}</td>
                <td>
                    <span class="badge ${getStatusBadgeClass(booking.status)}">
                        ${booking.status.toUpperCase()}
                    </span>
                </td>
            </tr>
        `).join('');
        
        container.classList.remove('d-none');
        
    } catch (error) {
        console.error('Error loading bookings:', error);
        loading.innerHTML = `
            <div class="alert alert-danger" role="alert">
                Failed to load bookings. Please refresh the page.
            </div>
        `;
    }
}

// Get badge class based on status
function getStatusBadgeClass(status) {
    switch (status.toLowerCase()) {
        case 'confirmed':
            return 'bg-primary';
        case 'completed':
            return 'bg-success';
        case 'cancelled':
            return 'bg-danger';
        default:
            return 'bg-secondary';
    }
}