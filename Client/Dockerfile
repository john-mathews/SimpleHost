# Use official Node.js image for Vite/React
FROM node:22-alpine

# Set working directory
WORKDIR /app

# Copy package.json and package-lock.json
COPY package.json package-lock.json ./

# If package.json does not exist, scaffold a new Vite React app
RUN if [ ! -f package.json ]; then \
    npm create vite@latest . -- --template react --yes; \
    npm install; \
  fi

RUN  npm install 

# Copy the rest of the app
COPY . .

# Expose port for Vite dev server
EXPOSE 5173

# Start Vite dev server
CMD ["npm", "run", "dev", "--", "--host"]
