// BarberBook - Booking Page JavaScript
// AI-generated booking form logic

const API_BASE_URL = '/api';

// Load barbers and services on page load
document.addEventListener('DOMContentLoaded', () => {
    loadBarbers();
    loadServicesForBooking();
    setMinDate();
    setupEventListeners();
});

// Set minimum date to today
function setMinDate() {
    const dateInput = document.getElementById('booking_date');
    const today = new Date().toISOString().split('T')[0];
    dateInput.setAttribute('min', today);
}

// Setup event listeners
function setupEventListeners() {
    const form = document.getElementById('booking-form');
    const dateInput = document.getElementById('booking_date');
    const barberSelect = document.getElementById('barber_id');
    
    // Load available slots when date or barber changes
    dateInput.addEventListener('change', loadAvailableSlots);
    barberSelect.addEventListener('change', loadAvailableSlots);
    
    // Handle form submission
    form.addEventListener('submit', handleBookingSubmit);
}

// Load barbers
async function loadBarbers() {
    const select = document.getElementById('barber_id');
    
    try {
        const response = await fetch(`${API_BASE_URL}/barbers`);
        
        if (!response.ok) {
            throw new Error('Failed to load barbers');
        }
        
        const barbers = await response.json();
        
        select.innerHTML = '<option value="">Choose a barber...</option>' +
            barbers.map(barber => 
                `<option value="${barber.barber_id}">${barber.barber_name} - ${barber.speciality}</option>`
            ).join('');
        
    } catch (error) {
        console.error('Error loading barbers:', error);
        select.innerHTML = '<option value="">Error loading barbers</option>';
    }
}

// Load services
async function loadServicesForBooking() {
    const select = document.getElementById('service');
    
    try {
        const response = await fetch(`${API_BASE_URL}/services`);
        
        if (!response.ok) {
            throw new Error('Failed to load services');
        }
        
        const services = await response.json();
        
        select.innerHTML = '<option value="">Choose a service...</option>' +
            services.map(service => 
                `<option value="${service.service_name}" data-price="${service.price}">
                    ${service.service_name} - €${parseFloat(service.price).toFixed(2)} (${service.duration_minutes} mins)
                </option>`
            ).join('');
        
    } catch (error) {
        console.error('Error loading services:', error);
        select.innerHTML = '<option value="">Error loading services</option>';
    }
}

// Load available time slots
async function loadAvailableSlots() {
    const barberId = document.getElementById('barber_id').value;
    const date = document.getElementById('booking_date').value;
    const timeSelect = document.getElementById('booking_time');
    
    if (!barberId || !date) {
        timeSelect.innerHTML = '<option value="">Select date and barber first</option>';
        return;
    }
    
    timeSelect.innerHTML = '<option value="">Loading slots...</option>';
    
    try {
        const response = await fetch(`${API_BASE_URL}/barbers/${barberId}/available-slots?date=${date}`);
        
        if (!response.ok) {
            throw new Error('Failed to load available slots');
        }
        
        const data = await response.json();
        const slots = data.available_slots || [];
        
        if (slots.length === 0) {
            timeSelect.innerHTML = '<option value="">No slots available</option>';
            return;
        }
        
        // Format time slots nicely
        timeSelect.innerHTML = '<option value="">Choose a time...</option>' +
            slots.map(slot => {
                const [hours, minutes] = slot.split(':');
                const hour = parseInt(hours);
                const ampm = hour >= 12 ? 'PM' : 'AM';
                const displayHour = hour > 12 ? hour - 12 : hour;
                return `<option value="${slot}">${displayHour}:${minutes} ${ampm}</option>`;
            }).join('');
        
    } catch (error) {
        console.error('Error loading available slots:', error);
        timeSelect.innerHTML = '<option value="">Error loading slots</option>';
    }
}

// Handle booking form submission
async function handleBookingSubmit(event) {
    event.preventDefault();
    
    const submitBtn = document.getElementById('submit-btn');
    const submitText = document.getElementById('submit-text');
    const submitSpinner = document.getElementById('submit-spinner');
    const messageDiv = document.getElementById('booking-message');
    
    // Disable submit button
    submitBtn.disabled = true;
    submitText.classList.add('d-none');
    submitSpinner.classList.remove('d-none');
    
    // Get form data
    const formData = {
        customer_name: document.getElementById('customer_name').value,
        customer_phone: document.getElementById('customer_phone').value,
        customer_email: document.getElementById('customer_email').value || null,
        barber_id: parseInt(document.getElementById('barber_id').value),
        booking_date: document.getElementById('booking_date').value,
        booking_time: document.getElementById('booking_time').value,
        service: document.getElementById('service').value,
        price: parseFloat(document.getElementById('service').selectedOptions[0].dataset.price)
    };
    
    try {
        const response = await fetch(`${API_BASE_URL}/bookings`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        });
        
        if (!response.ok) {
            throw new Error('Failed to create booking');
        }
        
        const booking = await response.json();
        
        // Show success message
        messageDiv.className = 'alert alert-success';
        messageDiv.textContent = `Booking confirmed! Your booking ID is ${booking.booking_id}. We'll see you on ${formData.booking_date} at ${formData.booking_time}.`;
        messageDiv.classList.remove('d-none');
        
        // Reset form
        document.getElementById('booking-form').reset();
        
        // Scroll to message
        messageDiv.scrollIntoView({ behavior: 'smooth', block: 'center' });
        
    } catch (error) {
        console.error('Error creating booking:', error);
        
        // Show error message
        messageDiv.className = 'alert alert-danger';
        messageDiv.textContent = 'Failed to create booking. Please try again or call us directly.';
        messageDiv.classList.remove('d-none');
        
    } finally {
        // Re-enable submit button
        submitBtn.disabled = false;
        submitText.classList.remove('d-none');
        submitSpinner.classList.add('d-none');
    }
}