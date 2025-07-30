Name:           grandchild
Version:        1.0
Release:        1
Summary:        A grandchild package
License:        MIT
Prefix:         /usr
Requires:       child

%description
This is a grandchild package that depends on child.

%install
mkdir -p %{buildroot}/%{_prefix}/bin
cat << EOF > %{buildroot}/%{_prefix}/bin/grandchild_command
#!/bin/sh
echo "Hello from grandchild"
EOF
chmod +x %{buildroot}/%{_prefix}/bin/grandchild_command

%files
%{_prefix}/bin/grandchild_command